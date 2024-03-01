import { jsonStringToLex, jsonToLex, Lexicons, ValidationError } from "@atproto/lexicon";
import Editor from "@monaco-editor/react";
import { useEffect, useState } from "react";
import { useMediaQuery } from "react-responsive";
import { lexicons } from "../../lexicons.js";
import { Badge } from "../catalyst/badge.js";
import { Text } from "../catalyst/text.js";

interface RecordEditorProps {
  lexID: string;
  lexRaw: string;
}

const lex = new Lexicons();

// Add all official lexicons to the lexicon registry
lexicons.forEach((lexicon) => {
  lex.add(jsonToLex(lexicon));
});

function RecordEditor({ lexID, lexRaw }: RecordEditorProps) {
  const [record, setRecord] = useState<string>(`{
    "$type": "app.bsky.feed.like",
    "createdAt": "2024-03-01T05:15:56.866Z",
    "subject": {
      "cid": "bafyreifjhu45jmyvpvrq5mtqhtjlxppguk2jmlz5q44kssv57hvqva4m7i",
      "uri": "at://did:plc:umdwh4r456zres7qrdbfpibd/app.bsky.feed.post/3kmlyenrxdh27"
    }
  }`);
  const [lexValidationResult, setLexValidationResult] = useState<string>("Record is Empty");

  const darkMode = useMediaQuery({
    query: "(prefers-color-scheme: dark)",
  });

  try {
    // Remove the lexicon if it's already registered
    lex.remove(lexID);
  } catch (e) { }

  // Register our custom lexicon
  lex.add(jsonStringToLex(lexRaw));

  function validateLexicon(raw: string): string {
    console.log("Validating Record");
    if (Object.keys(raw).length === 0) {
      return "Record is Empty";
    }

    try {
      lex.assertValidRecord(lexID, jsonStringToLex(raw));
    } catch (e) {
      if (e instanceof ValidationError) {
        console.log("Record Validation Error:", e);
        return e.message;
      }
      console.log("Record Parsing Error:", e);
      return "Failed to Parse Record: " + e;
    }
    return "Record is Valid";
  }

  function getBadgeColor(result: string): "green" | "yellow" | "red" {
    if (result === "Record is Valid") {
      return "green";
    } else if (
      result === "Unknown Collection" ||
      result === "Record is Empty"
    ) {
      return "yellow";
    } else {
      return "red";
    }
  }

  useEffect(() => {
    setLexValidationResult(validateLexicon(record));
  }, [record, lexRaw]);

  return (
    <div className="flex min-h-0 grow flex-col pt-12 lg:basis-0">
      <Text className="mb-4 text-center">
        <span className="text-3xl dark:text-slate-100">
          Validate a Record
        </span>
      </Text>

      <div className="h-96 grow lg:h-auto">
        <Editor
          width="100%"
          height="100%"
          language="json"
          theme={darkMode ? "vs-dark" : "vs-light"}
          value={record}
          options={{
            readOnly: false,
            wordWrap: "on",
            lineNumbersMinChars: 3,
          }}
          onChange={(value) => { if (value) setRecord(value) }}
        />
      </div>

      <div className="mt-2">
        <Badge color={getBadgeColor(lexValidationResult)}>{lexValidationResult}</Badge>
      </div>
    </div>
  );
}

export default RecordEditor;
