import { Lexicons, jsonStringToLex } from "@atproto/lexicon";
import { Editor } from "@monaco-editor/react";
import { FC, useEffect, useState } from "react";
import { useMediaQuery } from "react-responsive";
import { lexicons } from "../../lexicons.ts";
import { Badge } from "../catalyst/badge.tsx";
import { Text } from "../catalyst/text.js";
import RecordEditor from "./RecordEditor.tsx";

const knownLexicons: string[] = [];

let initialLexicon: any = null;

lexicons.forEach((lexicon) => {
  if (lexicon.defs.main?.type === "record") {
    knownLexicons.push(lexicon.id);
    if (lexicon.id === "app.bsky.feed.like") {
      initialLexicon = lexicon
    }
  }
});

interface LexPair {
  lexID: string;
  lexRaw: string;
}

const LexiconEditor: FC<{}> = () => {
  const [activeLexPair, setActiveLexPair] = useState<LexPair>({
    lexID: "app.bsky.feed.like",
    lexRaw: JSON.stringify(initialLexicon, null, 2),
  });

  useEffect(() => {
    document.title = "Edit Lexicons";
  }, []);


  return (
    <div className="flex min-h-0 min-w-0 grow flex-col gap-6 p-4 pt-6 lg:h-dvh lg:flex-row">
      <div
        className="flex min-h-0 grow flex-col gap-2 lg:basis-0 dark:[color-scheme:dark]"
      >
        <LexEditor
          activeLexPair={activeLexPair}
          setActiveLexPair={setActiveLexPair}
        />
      </div>
      <RecordEditor lexID={activeLexPair.lexID} lexRaw={activeLexPair.lexRaw} />
    </div>
  );
};

export default LexiconEditor;

interface LexEditorProps {
  activeLexPair: LexPair;
  setActiveLexPair: (activeLexPair: LexPair) => void;
}

function LexEditor({ activeLexPair, setActiveLexPair }: LexEditorProps) {
  const [pendingLex, setPendingLex] = useState<string>(activeLexPair.lexRaw);
  const [lexValidationResult, setLexValidationResult] = useState<string>("Lexicon is Empty");

  const darkMode = useMediaQuery({
    query: "(prefers-color-scheme: dark)",
  });

  const lex = new Lexicons();

  function validateLexicon(raw: string): string {
    if (raw.length === 0) {
      return "Lexicon is Empty";
    }

    let asLex;

    try {
      asLex = jsonStringToLex(raw);
      try {
        lex.remove(asLex.id);
      } catch { }
      lex.add(asLex);
    } catch (e) {
      console.log("Lexicon is Invalid:", e);
      return "Lexicon is Invalid: " + e;
    }

    setActiveLexPair({ lexID: asLex.id, lexRaw: raw });
    return "Lexicon is Valid";
  }

  function getBadgeColor(result: string): "green" | "yellow" | "red" {
    if (result === "Lexicon is Valid") {
      return "green";
    } else if (
      result === "Lexicon is Empty"
    ) {
      return "yellow";
    } else {
      return "red";
    }
  }

  useEffect(() => {
    setLexValidationResult(validateLexicon(pendingLex));
  }, [pendingLex]);

  return (
    <div className="flex min-h-0 grow flex-col pt-12 lg:basis-0">
      <Text className="mb-2 text-center">
        <span className="text-3xl">
          Build a Lexicon
        </span>
      </Text>
      <div className="h-96 grow lg:h-auto">
        <Editor
          width="100%"
          height="100%"
          language="json"
          theme={darkMode ? "vs-dark" : "vs-light"}
          value={pendingLex}
          options={{
            readOnly: false,
            wordWrap: "on",
            lineNumbersMinChars: 3,
          }}
          onChange={(value) => {
            if (value) {
              setPendingLex(value);
            }
          }}
        />
      </div>
      <div className="mt-2">
        <Badge color={getBadgeColor(lexValidationResult)}>{lexValidationResult}</Badge>
      </div>
    </div>
  )
}


