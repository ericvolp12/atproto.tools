# AT Proto Looking Glass

A collection of looking-glass tools for the AT Proto Network

## Services

### Consumer

The Looking Glass Consumer is a Go service that connects to an AT Proto Firehose and listens for events.

When an event is received, the consumer will attempt to unpack any records from the event and add them to a SQLite database for later querying.

Event metadata is also added to the database, including the event's timestamp, the event's sequence number, and the event's type.

The consumer deletes old records from the database to keep the database from growing too large by default.

#### Running the Consumer

To run the consumer via Docker Compose, you can run: `make lg-consumer-up`.

The consumer stores its SQLite DB in `./data/lg-consumer` by default.

## Tools

### Checkout

The Checkout tool lets you download your AT Proto repo as a directory of JSON files (one per record).

It lets you select a PDS to download from, defaulting to the Relay (`bsky.network`) and supports compressing the result into a gzipped tarball (since lots of this JSON data is highly compressible).

To use the Checkout tool, you can `go run cmd/checkout/main.go <repo-DID>`.

Use the `--help` flag for more options.
