import { JSONRecord } from '../../models/Record'
import { Button } from '../catalyst/button'
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../catalyst/dialog'
import { Lexicons, ValidationError, jsonToLex } from '@atproto/lexicon';
import { lexicons } from '../../lexicons.ts';
import { Badge } from '../catalyst/badge.tsx';
import Editor, { DiffEditor, useMonaco, loader } from '@monaco-editor/react';

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
    isOpen: boolean
    setIsOpen: (isOpen: boolean) => void
}

function RawRecord({ record, isOpen, setIsOpen }: RawRecordProps) {
    function validateLexicon(collection: string, raw: any): string {
        if (!knownLexicons.includes(collection)) {
            return 'Unknown Collection'
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
        } else if (result === 'Unknown Collection') {
            return 'yellow'

        } else {
            return 'red'
        }
    }

    if (!record || !record.raw) return null

    const lexValidationResult = validateLexicon(record.collection, record.raw)
    const badgeColor = getBadgeColor(lexValidationResult)

    const formattedRaw = JSON.stringify(record.raw, null, 2)
    const numLines = formattedRaw.split('\n').length

    return (
        <Dialog open={isOpen} onClose={setIsOpen} size="3xl">
            <DialogTitle>Raw Record Viewer</DialogTitle>
            <DialogDescription>
                Raw record content for: <span className="text-sm font-mono">at://{record.repo}/{record.collection}/{record.rkey}</span>
            </DialogDescription>
            <DialogBody className="min-w-full">
                <Editor
                    width="100%"
                    height={`${numLines * 1.5}rem`}
                    language="json"
                    theme="vs-dark"
                    value={formattedRaw}
                    options={{ readOnly: true, wordWrap: 'on' }}
                />
            </DialogBody>
            <DialogActions className='justify-between'>
                <Badge color={badgeColor}>{lexValidationResult}</Badge>
                <Button onClick={() => setIsOpen(false)}>Close</Button>
            </DialogActions>
        </Dialog>
    )
}

export default RawRecord
