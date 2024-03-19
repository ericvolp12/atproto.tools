package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/data"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/repo"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.App{
		Name:    "checkout",
		Usage:   "atproto repo checkout",
		Version: "0.0.3",
	}

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "pds-host",
			Usage:   "host of the PDS or Relay to fetch the repo from (with protocol)",
			Value:   "https://bsky.network",
			EnvVars: []string{"PDS_URL"},
		},
		&cli.StringFlag{
			Name:    "output-dir",
			Usage:   "directory to write the repo to",
			Value:   "./out/<repo-did>",
			EnvVars: []string{"OUTPUT_DIR"},
		},
		&cli.BoolFlag{
			Name:  "compress",
			Usage: "compress the resulting directory into a gzip file",
		},
	}

	app.ArgsUsage = "<repo-did>"

	app.Action = Checkout

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func Checkout(cctx *cli.Context) error {
	ctx := cctx.Context
	rawDID := cctx.Args().First()

	did, err := syntax.ParseDID(rawDID)
	if err != nil {
		log.Println("Error parsing DID", err)
		return fmt.Errorf("Error parsing DID: %v", err)
	}

	url := fmt.Sprintf("%s/xrpc/com.atproto.sync.getRepo?did=%s", cctx.String("pds-host"), did.String())

	outputDir := cctx.String("output-dir")
	compress := cctx.Bool("compress")

	if outputDir == "./out/<repo-did>" {
		outputDir = fmt.Sprintf("./out/%s", did.String())
		outputDir, err = filepath.Abs(outputDir)
		if err != nil {
			log.Println("Error getting absolute path", err)
			return fmt.Errorf("Error getting absolute path: %v", err)
		}

		if !compress {
			// Create the directory if it doesn't exist and in uncompressed mode
			err = os.MkdirAll(outputDir, 0755)
			if err != nil {
				log.Println("Error creating directory", err)
				return fmt.Errorf("Error creating directory: %v", err)
			}
		}
	}

	// Initialize HTTP client
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Println("Error creating request", err)
		return fmt.Errorf("Error creating request: %v", err)
	}

	req.Header.Set("Accept", "application/vnd.ipld.car")
	req.Header.Set("User-Agent", fmt.Sprintf("atproto.tools.checkout/%s", cctx.App.Version))

	log.Println("Fetching repo", "DID", did.String(), "URL", url)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request", err)
		return fmt.Errorf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Error response", "status", resp.StatusCode)
		return fmt.Errorf("Error response: %v", resp.StatusCode)
	}

	r, err := repo.ReadRepoFromCar(ctx, resp.Body)
	if err != nil {
		log.Println("Error reading repo", err)
		return fmt.Errorf("Error reading repo: %v", err)
	}

	var tarWriter *tar.Writer
	var gzipWriter *gzip.Writer
	var tarFile *os.File

	if compress {
		// Create the tar.gz file
		tarGzPath := filepath.Join(outputDir + ".tar.gz")
		tarFile, err = os.Create(tarGzPath)
		if err != nil {
			log.Println("Error creating tar.gz file", err)
			return fmt.Errorf("Error creating tar.gz file: %v", err)
		}
		defer tarFile.Close()

		gzipWriter = gzip.NewWriter(tarFile)
		defer gzipWriter.Close()

		tarWriter = tar.NewWriter(gzipWriter)
		defer tarWriter.Close()
	}

	numRecords := 0
	collectionsSeen := make(map[string]struct{})

	err = r.ForEach(ctx, "", func(path string, nodeCid cid.Cid) error {
		recordCid, rec, err := r.GetRecordBytes(ctx, path)
		if err != nil {
			log.Println("Error getting record", err)
			return nil
		}

		// Verify that the record CID matches the node CID
		if recordCid != nodeCid {
			log.Println("Mismatch in record and node CID", "recordCID", recordCid, "nodeCID", nodeCid)
			return nil
		}

		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			log.Println("Path does not have 2 parts", "path", path)
			return nil
		}

		collection := parts[0]
		rkey := parts[1]

		numRecords++
		if _, ok := collectionsSeen[collection]; !ok {
			collectionsSeen[collection] = struct{}{}
		}

		asCbor, err := data.UnmarshalCBOR(*rec)
		if err != nil {
			log.Println("Error unmarshalling record", err)
			return fmt.Errorf("Failed to unmarshal record: %w", err)
		}

		recJSON, err := json.Marshal(asCbor)
		if err != nil {
			log.Println("Error marshalling record to JSON", err)
			return fmt.Errorf("Failed to marshal record to JSON: %w", err)
		}

		if compress {
			// Write the record directly to the tar.gz file
			hdr := &tar.Header{
				Name: fmt.Sprintf("%s/%s.json", collection, rkey),
				Mode: 0600,
				Size: int64(len(recJSON)),
			}
			if err := tarWriter.WriteHeader(hdr); err != nil {
				log.Println("Error writing tar header", err)
				return err
			}
			if _, err := tarWriter.Write(recJSON); err != nil {
				log.Println("Error writing record to tar file", err)
				return err
			}
		} else {
			// Write the record to a file in uncompressed mode
			recordPath := filepath.Join(outputDir, collection, fmt.Sprintf("%s.json", rkey))
			err = os.MkdirAll(filepath.Dir(recordPath), 0755)
			if err != nil {
				log.Println("Error creating collection directory", err)
				return nil // Continue processing other records
			}
			err = os.WriteFile(recordPath, recJSON, 0644)
			if err != nil {
				log.Println("Error writing record to file", err)
				return nil // Continue processing other records
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Error during ForEach", err)
		return fmt.Errorf("Error during ForEach: %v", err)
	}

	log.Println("Checkout complete", "Output directory", outputDir, "Number of records", numRecords, "Number of collections", len(collectionsSeen))

	return nil
}
