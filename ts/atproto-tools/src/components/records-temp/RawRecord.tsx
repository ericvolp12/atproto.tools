import { JSONRecord } from '../../models/Record'
import { Button } from '../catalyst/button'
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../catalyst/dialog'
import { Lexicons, ValidationError, jsonToLex } from '@atproto/lexicon';
import { lexicons } from '../../lexicons.ts';
import { Badge } from '../catalyst/badge.tsx';
import Editor, { DiffEditor, useMonaco, loader } from '@monaco-editor/react';
import { Text } from '../catalyst/text.tsx';

const lex = new Lexicons()
const knownLexicons: string[] = []
lexicons.forEach((lexicon) => {
    if (lexicon.defs.main?.type === 'record') {
        // @ts-ignore
        lex.add(lexicon)
        knownLexicons.push(lexicon.id)
    }
})


interface RawRecordProps {
    record: JSONRecord
}

function RawRecord({ record }: RawRecordProps) {
    function validateLexicon(collection: string, raw: any): string {
        if (!knownLexicons.includes(collection)) {
            return 'Unknown Collection'
        }

        if (Object.keys(raw).length === 0) {
            return 'Record is Empty'
        }

        try {
            lex.assertValidRecord(collection, jsonToLex(raw))
        } catch (e) {
            if (e instanceof ValidationError) {
                console.log(e);
                return e.message
            }
        }
        return 'Record is Valid'
    }

    function getBadgeColor(result: string): "green" | "yellow" | "red" {
        if (result === 'Record is Valid') {
            return 'green'
        } else if (result === 'Unknown Collection' || result === 'Record is Empty') {
            return 'yellow'
        } else {
            return 'red'
        }
    }

    if (!record) return null
    if (!record.raw) {
        record.raw = {}
    }

    const lexValidationResult = validateLexicon(record.collection, record.raw)
    const badgeColor = getBadgeColor(lexValidationResult)

    const formattedRaw = JSON.stringify(record.raw, null, 2)
    let numLines = formattedRaw.split('\n').length
    if (numLines < 5) numLines = 5
    if (numLines > 25) numLines = 25

    return (
        <div className='flex grow flex-col min-h-0 pt-12 lg:basis-0'>
            <Text className="mb-2">
                Raw record content for: <span className="text-sm font-mono">at://{record.repo}/{record.collection}/{record.rkey}</span>
            </Text>

            <div className='grow h-96 lg:h-auto'>
                <Editor
                    width="100%"
                    height="100%"
                    language="json"
                    theme="vs-dark"
                    value={formattedRaw}
                    options={{ readOnly: true, wordWrap: 'on', lineNumbersMinChars: 3 }}
                />
            </div>

            <div className="mt-2">
                <Badge color={badgeColor}>{lexValidationResult}</Badge>
            </div>
        </div>
    )
}

export default RawRecord
