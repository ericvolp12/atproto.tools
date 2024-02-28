import { JSONRecord } from '../../models/Record'
import { Button } from '../catalyst/button'
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../catalyst/dialog'
import JsonView from '@uiw/react-json-view';
import { nordTheme } from '@uiw/react-json-view/nord';


interface RawRecordProps {
    record: JSONRecord
    isOpen: boolean
    setIsOpen: (isOpen: boolean) => void
}

function RawRecord({ record, isOpen, setIsOpen }: RawRecordProps) {
    return (
        (record && record?.raw) &&
        <Dialog open={isOpen} onClose={setIsOpen} size="3xl">
            <DialogTitle>Raw Record Viewer</DialogTitle>
            <DialogDescription>
                Raw record content for: <pre className="text-sm">at://{record.repo}/{record.collection}/{record.rkey}</pre>
            </DialogDescription>
            <DialogBody className="min-w-full">
                <JsonView value={record?.raw} style={nordTheme} shortenTextAfterLength={40} />
            </DialogBody>
            <DialogActions>
                <Button onClick={() => setIsOpen(false)}>Close</Button>
            </DialogActions>
        </Dialog>

    )
}

export default RawRecord
