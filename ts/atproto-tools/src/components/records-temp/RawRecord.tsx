import { JSONRecord } from '../../models/Record'
import { Button } from '../catalyst/button'
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../catalyst/dialog'
import JsonView from '@uiw/react-json-view';
import { nordTheme } from '@uiw/react-json-view/nord';
import { Lexicons, ValidationError } from '@atproto/lexicon';
import { lexicons } from '../../lexicons.ts';
import { Badge } from '../catalyst/badge.tsx';

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
            lex.assertValidRecord(collection, raw)
        } catch (e) {
            if (e instanceof ValidationError) {
                return e.message
            }
        }
        return 'Record is Valid'
    }

    function badgeColor(collection: string, raw: any): "green" | "yellow" | "red" {
        if (validateLexicon(collection, raw) === 'Record is Valid') {
            return 'green'
        } else if (validateLexicon(collection, raw) === 'Unknown Collection') {
            return 'yellow'

        } else {
            return 'red'
        }
    }

    return (
        (record && record?.raw) &&
        <Dialog open={isOpen} onClose={setIsOpen} size="3xl">
            <DialogTitle>Raw Record Viewer</DialogTitle>
            <DialogDescription>
                Raw record content for: <span className="text-sm font-mono">at://{record.repo}/{record.collection}/{record.rkey}</span>
            </DialogDescription>
            <DialogBody className="min-w-full">
                <JsonView
                    value={record?.raw}
                    style={nordTheme}
                    shortenTextAfterLength={300}
                    displayDataTypes={false}
                    className="rounded-md p-2 w-full"
                    enableClipboard={true}
                />
            </DialogBody>
            <DialogActions className='justify-between'>
                <Badge color={badgeColor(record.collection, record.raw)}>{validateLexicon(record.collection, record.raw)}</Badge>
                <Button onClick={() => setIsOpen(false)}>Close</Button>
            </DialogActions>
        </Dialog>

    )
}

export default RawRecord
