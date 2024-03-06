import { FC, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { useSearchParams } from "react-router-dom";

import { LOOKING_GLASS_HOST } from "../../constants";
import { JSONRecord } from "../../models/Record";
import { Button } from "../catalyst/button";
import { Field, FieldGroup, Fieldset, Label } from "../catalyst/fieldset";
import { Input } from "../catalyst/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../catalyst/table";
import RawRecord from "./RawRecord";

const Records: FC<{}> = () => {
  const [selectedRecord, setSelectedRecord] = useState<JSONRecord | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [didQuery, setDIDQuery] = useState<string | null>(null);
  const [collectionQuery, setCollectionQuery] = useState<string | null>(null);
  const [rkeyQuery, setRKeyQuery] = useState<string | null>(null);
  const [seqQuery, setSeqQuery] = useState<string | null>(null);

  const [searchParams, setSearchParams] = useSearchParams();

  useEffect(() => {
    document.title = "View Firehose Records";
  }, []);

  const fetchRecords = async (): Promise<JSONRecord[]> => {
    const url = new URL(`${LOOKING_GLASS_HOST}/records`);
    if (didQuery) url.searchParams.append("did", didQuery);
    if (collectionQuery) url.searchParams.append("collection", collectionQuery);
    if (rkeyQuery) url.searchParams.append("rkey", rkeyQuery);
    if (seqQuery) url.searchParams.append("seq", seqQuery);

    const response = await fetch(url.toString());
    const data = await response.json();

    if (data.error) {
      setError(data.error);
      return [];
    } else {
      return data.records.map((record: JSONRecord) => ({
        ...record,
        key: `${record.seq}_${record.collection}_${record.rkey}`,
      }));
    }
  };

  const { data: records, isLoading } = useQuery(
    ["records", didQuery, collectionQuery, rkeyQuery, seqQuery],
    fetchRecords,
  );

  useEffect(() => {
    if (records?.length) {
      setSelectedRecord(records.find((record) => record.raw) || null);
    }
  }, [records]);

  useEffect(() => {
    setDIDQuery(searchParams.get("did") || null);
    setCollectionQuery(searchParams.get("collection") || null);
    setRKeyQuery(searchParams.get("rkey") || null);
    setSeqQuery(searchParams.get("seq") || null);

    if (searchParams.has("uri")) {
      const uri = searchParams.get("uri")!;
      if (uri.startsWith("at://")) {
        const [, , did, collection, rkey] = uri.split("/");
        setDIDQuery(did);
        setCollectionQuery(collection);
        setRKeyQuery(rkey);
      }
    }
  }, [searchParams]);

  return (
    <div className="flex min-h-0 min-w-0 grow flex-col gap-6 p-4 pt-6 lg:h-dvh lg:flex-row">
      <div className="flex min-h-0 grow flex-col gap-2 lg:basis-0 dark:[color-scheme:dark]">
        <h1 className="text-center text-4xl font-bold">
          View Firehose Records
        </h1>
        <SearchForm
          didQuery={didQuery}
          collectionQuery={collectionQuery}
          rkeyQuery={rkeyQuery}
          seqQuery={seqQuery}
          setSearchParams={setSearchParams}
        />

        <div className="h-96 min-h-0 grow overflow-y-auto lg:h-auto lg:overflow-x-hidden">
          <RecordsTable
            records={records || []}
            setSelectedRecord={setSelectedRecord}
            selectedRecord={selectedRecord}
            isLoading={isLoading}
          />
        </div>
      </div>
      <RawRecord
        record={selectedRecord!}
        key={records ? "loaded" : "not yet loaded"}
      />
    </div>
  );
};

interface RecordsTableProps {
  records: JSONRecord[];
  selectedRecord: JSONRecord | null;
  setSelectedRecord: (record: JSONRecord) => void;
  isLoading: boolean;
}

const RecordsTable: FC<RecordsTableProps> = ({
  records,
  selectedRecord,
  setSelectedRecord,
  isLoading,
}) => {
  if (isLoading) {
    return <div>Loading...</div>;
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTableElement>) => {
    e.preventDefault();
    if (e.key === "ArrowDown" && selectedRecord) {
      const index = records.findIndex((record) => record.key === selectedRecord.key);
      if (index < records.length - 1) {
        setSelectedRecord(records[index + 1]);
        scrollToSelectedRecord(records[index + 1].key || "");
      }
    }
    if (e.key === "ArrowUp" && selectedRecord) {
      const index = records.findIndex((record) => record.key === selectedRecord.key);
      if (index > 0 && records.length > 0) {
        setSelectedRecord(records[index - 1]);
        scrollToSelectedRecord(records[index - 1].key || "");
      }
    }
  };

  const scrollToSelectedRecord = (key: string) => {
    const selectedElement = document.getElementById(key);
    selectedElement?.scrollIntoView({ block: "nearest" });
  };

  return (
    <Table
      striped
      dense
      grid
      sticky
      className="mx-0 [--gutter:theme(spacing.2)] focus:outline-none sm:[--gutter:theme(spacing.2)]"
      tabIndex={0}
      onKeyDown={handleKeyDown}
    >
      <TableHead>
        <TableRow>
          <TableHeader>Seq</TableHeader>
          <TableHeader>Repo</TableHeader>
          <TableHeader>Collection</TableHeader>
          <TableHeader>Record Key</TableHeader>
          <TableHeader>Action</TableHeader>
        </TableRow>
      </TableHead>
      <TableBody>
        {records.map((record) => (
          <TableRow
            key={record?.key || ""}
            id={record?.key || ""}
            className={
              (selectedRecord?.key === record?.key
                ? "!bg-zinc-950/[15%] dark:!bg-white/[15%] "
                : "") + " scroll-m-36"
            }
            onClick={() => setSelectedRecord(record)}
          >
            <TableCell className="font-mono text-zinc-400">
              <Tooltip text={record.pds || ""} position="right">
                <span>{record.seq}</span>
              </Tooltip>
            </TableCell>
            <TableCell className="font-mono">
              <Tooltip text={record.handle || ""} position="top">
                <span>{record.repo}</span>
              </Tooltip>
            </TableCell>
            <TableCell className="text-zinc-400">{record.collection}</TableCell>
            <TableCell className="font-mono">{record.rkey}</TableCell>
            <TableCell className="text-zinc-400">{record.action}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};

interface SearchFormProps {
  didQuery: string | null;
  collectionQuery: string | null;
  rkeyQuery: string | null;
  seqQuery: string | null;
  setSearchParams: (searchParams: URLSearchParams) => void;
}

const SearchForm: FC<SearchFormProps> = ({ didQuery, collectionQuery, rkeyQuery, seqQuery, setSearchParams }) => {
  const [didSearch, setDIDSearch] = useState<string | null>(didQuery);
  const [collectionSearch, setCollectionSearch] = useState<string | null>(collectionQuery);
  const [rkeySearch, setRKeySearch] = useState<string | null>(rkeyQuery);
  const [seqSearch, setSeqSearch] = useState<string | null>(seqQuery);

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const searchParams = new URLSearchParams();
    if (didSearch) searchParams.append("did", didSearch);
    if (collectionSearch) searchParams.append("collection", collectionSearch);
    if (rkeySearch) searchParams.append("rkey", rkeySearch);
    if (seqSearch) searchParams.append("seq", seqSearch);
    setSearchParams(searchParams);
  };

  return (
    <form onSubmit={handleSearch}>
      <Fieldset className="mb-4">
        <FieldGroup>
          <div className="grid grid-cols-1 gap-8 sm:grid-cols-9 sm:gap-4">
            <Field className="col-span-1">
              <Label>Seq</Label>
              <Input name="seq" value={seqSearch || ""} onChange={(e) => setSeqSearch(e.target.value.trim())} />
            </Field>
            <Field className="col-span-3">
              <Label>DID</Label>
              <Input name="did" value={didSearch || ""} onChange={(e) => setDIDSearch(e.target.value.trim())} />
            </Field>
            <Field className="col-span-2">
              <Label>Collection</Label>
              <Input
                name="collection"
                value={collectionSearch || ""}
                onChange={(e) => setCollectionSearch(e.target.value.trim())}
                disabled={!didSearch}
              />
            </Field>
            <Field className="col-span-2">
              <Label>Record Key</Label>
              <Input
                name="rkey"
                value={rkeySearch || ""}
                onChange={(e) => setRKeySearch(e.target.value.trim())}
                disabled={!didSearch || !collectionSearch}
              />
            </Field>
            <div className="mt-auto justify-self-end">
              <Button type="submit">Search</Button>
            </div>
          </div>
        </FieldGroup>
      </Fieldset>
    </form>
  );
}

type PositionKey = "top" | "bottom" | "left" | "right";

const positionStyles: { [key in PositionKey]: string } = {
  top: "bottom-[calc(100%+0.5rem)] left-[38%] -translate-x-[50%]",
  bottom: "top-[calc(100%+0.5rem)] left-[38%] -translate-x-[50%]",
  left: "right-[calc(100%+0.5rem)] top-[38%] -translate-y-[50%]",
  right: "left-[calc(100%+0.5rem)] top-[38%] -translate-y-[50%]",
};

const positionSVGs: { [key in PositionKey]: JSX.Element } = {
  top: (
    <svg
      className="absolute left-0 top-full h-2 w-full text-black"
      x="0px"
      y="0px"
      viewBox="0 0 255 255"
    >
      <polygon className="fill-current" points="0,0 127.5,127.5 255,0" />
    </svg>
  ),
  bottom: (
    <svg
      className="absolute bottom-full left-0 h-2 w-full text-black"
      x="0px"
      y="0px"
      viewBox="0 0 255 255"
    >
      <polygon className="fill-current" points="0,0 127.5,127.5 255,0" />
    </svg>
  ),
  left: (
    <svg
      className="absolute left-full top-0 h-full w-2 text-black"
      x="0px"
      y="0px"
      viewBox="0 0 255 255"
    >
      <polygon className="fill-current" points="0,0 127.5,127.5 0,255" />
    </svg>
  ),
  right: (
    <svg
      className="absolute right-full top-0 h-full w-2 text-black"
      x="0px"
      y="0px"
      viewBox="0 0 255 255"
    >
      <polygon className="fill-current" points="255,0 0,127.5 255,255" />
    </svg>
  ),
};

function Tooltip({
  children,
  text,
  position,
}: {
  children: React.ReactNode;
  text: string;
  position: PositionKey;
}) {
  const [showTooltip, setShowTooltip] = useState(false);

  return (
    <div
      className="group relative"
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      <div
        className={`absolute z-30 ${positionStyles[position]} hidden w-auto transition duration-300 ease-in-out group-hover:block ${!showTooltip ? "hidden opacity-0" : "opacity-100"
          }`}
      >
        <div className="bottom-full right-0 whitespace-nowrap rounded bg-black px-4 py-1 text-xs text-white">
          {text}
          {positionSVGs[position]}
        </div>
      </div>
      {children}
    </div>
  );
}

export default Records;
