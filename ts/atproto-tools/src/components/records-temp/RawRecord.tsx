import { jsonToLex, Lexicons, ValidationError } from "@atproto/lexicon";
import Editor from "@monaco-editor/react";

import { lexicons } from "../../lexicons.ts";
import { JSONRecord } from "../../models/Record";
import { Badge } from "../catalyst/badge.tsx";
import { Text } from "../catalyst/text.tsx";
import { useState } from "react";
import { useMediaQuery } from "react-responsive";


const lex = new Lexicons();
const knownLexicons: string[] = [];
lexicons.forEach((lexicon) => {
  if (lexicon.defs.main?.type === "record") {
    // @ts-ignore
    lex.add(lexicon);
    knownLexicons.push(lexicon.id);
  }
});

interface RawRecordProps {
  record: JSONRecord;
}

function RawRecord({ record }: RawRecordProps) {
  const darkMode = useMediaQuery({
    query: "(prefers-color-scheme: dark)",
  });

  function validateLexicon(collection: string, raw: any): string {
    if (!knownLexicons.includes(collection)) {
      return "Unknown Collection";
    }

    if (Object.keys(raw).length === 0) {
      return "Record is Empty";
    }

    try {
      lex.assertValidRecord(collection, jsonToLex(raw));
    } catch (e) {
      if (e instanceof ValidationError) {
        console.log(e);
        return e.message;
      }
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

  if (!record) {
    record = {
      repo: "",
      collection: "",
      rkey: "",
      seq: 0,
      action: "",
      raw: {},
    };
  }
  if (!record.raw) {
    record.raw = {};
  }

  const lexValidationResult = validateLexicon(record.collection, record.raw);
  const badgeColor = getBadgeColor(lexValidationResult);

  const formattedRaw = JSON.stringify(record.raw, null, 2);
  let numLines = formattedRaw.split("\n").length;
  if (numLines < 5) numLines = 5;
  if (numLines > 25) numLines = 25;

  return (
    <div className="flex min-h-0 grow flex-col pt-12 lg:basis-0">
      <Text className="mb-2">
        {record.collection !== "" ? (
          <>
            Raw record content for:{" "}
            <span className="break-all font-mono text-sm">
              at://{record.repo}/{record.collection}/{record.rkey}
            </span>
          </>
        ) : (
          <span className="break-all font-mono text-sm">
            No record selected
          </span>
        )}
      </Text>

      <div className="h-96 grow lg:h-auto">
        <Editor
          width="100%"
          height="100%"
          language="json"
          theme={darkMode ? "vs-dark" : "vs-light"}
          value={formattedRaw}
          options={{
            readOnly: true,
            wordWrap: "on",
            lineNumbersMinChars: 3,
          }}
        />
      </div>

      <div className="mt-2">
        <Badge color={badgeColor}>{lexValidationResult}</Badge>
      </div>
    </div>
  );
}

export default RawRecord;
