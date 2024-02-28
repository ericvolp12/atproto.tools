import { FC, useEffect, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../catalyst/table'
import { JSONRecord } from '../../models/Record'
import { LOOKING_GLASS_HOST } from "../../constants";


const Records: FC<{}> = () => {
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
                <div className="mt-8 ">
                    <RecordsTable records={records} />
                </div>
            </div>
        </div >
    );
};

function RecordsTable({ records }: { records: JSONRecord[] }) {
    return (
        <Table striped dense grid className="[--gutter:theme(spacing.6)] sm:[--gutter:theme(spacing.8)]">
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
                    <TableRow key={`${record.seq}_${record.collection}_${record.rkey}`}>
                        <TableCell>{record.seq}</TableCell>
                        <TableCell>{record.repo}</TableCell>
                        <TableCell className="text-zinc-500">{record.collection}</TableCell>
                        <TableCell>{record.rkey}</TableCell>
                        <TableCell className="text-zinc-500">{record.action}</TableCell>
                    </TableRow>
                ))}
            </TableBody>
        </Table>
    )
}

export default Records;
