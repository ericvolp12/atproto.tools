import { FC, useEffect, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../catalyst/table'
import { JSONRecord } from '../../models/Record'
import { LOOKING_GLASS_HOST } from "../../constants";
import { Button } from "../catalyst/button";
import RawRecord from "./RawRecord";


const Records: FC<{}> = () => {
    const [selectedRecord, setSelectedRecord] = useState<JSONRecord | null>(null);
    const [isRawRecordOpen, setIsRawRecordOpen] = useState(false);
    const [records, setRecords] = useState<JSONRecord[]>([]);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        document.title = "View Firehose Records";
    }, []);

    useEffect(() => {
        fetch(`${LOOKING_GLASS_HOST}/records`)
            .then((response) => response.json())
            .then((data) => {
                if (data.error) {
                    setError(data.error);
                } else {
                    setRecords(data.records);
                }
            });
    }, []);

    return (
        <div className="mt-6">
            <div className="mx-auto max-w-7xl px-2 align-middle">
                <h1 className="text-4xl font-bold">View Firehose Records</h1>
                <div className="mt-8">
                    <RawRecord record={selectedRecord!} isOpen={isRawRecordOpen} setIsOpen={setIsRawRecordOpen} />
                    <RecordsTable records={records} setSelectedRecord={setSelectedRecord} setIsRawRecordOpen={setIsRawRecordOpen} />
                </div>
            </div>
        </div >
    );
};

function RecordsTable({ records, setSelectedRecord, setIsRawRecordOpen }: {
    records: JSONRecord[],
    setSelectedRecord: (record: JSONRecord) => void,
    setIsRawRecordOpen: (isOpen: boolean) => void
}) {
    return (
        <Table striped dense grid className="[--gutter:theme(spacing.6)] sm:[--gutter:theme(spacing.8)]">
            <TableHead>
                <TableRow>
                    <TableHeader>Seq</TableHeader>
                    <TableHeader>Repo</TableHeader>
                    <TableHeader>Collection</TableHeader>
                    <TableHeader>Record Key</TableHeader>
                    <TableHeader>Action</TableHeader>
                    <TableHeader>Record</TableHeader>
                </TableRow>
            </TableHead>
            <TableBody>
                {records.map((record) => (
                    <TableRow key={`${record.seq}_${record.collection}_${record.rkey}`}>
                        <TableCell className="font-mono text-zinc-400">{record.seq}</TableCell>
                        <TableCell className="font-mono">{record.repo}</TableCell>
                        <TableCell className="text-zinc-400">{record.collection}</TableCell>
                        <TableCell className="font-mono">{record.rkey}</TableCell>
                        <TableCell className="text-zinc-400">{record.action}</TableCell>
                        <TableCell>
                            {record.raw && (
                                <Button className="w-12 h-6 text-xs" onClick={() => {
                                    setSelectedRecord(record);
                                    setIsRawRecordOpen(true);
                                }}>View</Button>
                            )}
                        </TableCell>
                    </TableRow>
                ))}
            </TableBody>
        </Table>
    )
}


export default Records;
