export declare interface JSONRecord {
  seq: number;
  repo: string;
  collection: string;
  rkey: string;
  action: string;
  raw?: { [key: string]: any };
}

export declare interface RecordsResponse {
  records: JSONRecord[];
  error?: string;
}
