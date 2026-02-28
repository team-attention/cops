import { useState } from 'react'
import { useIssueAPIKey } from '../hook/use-issue-api-key'
import type { FormEventHandler } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/gen/shadcn/ui/dialog'
import { Button } from '@/gen/shadcn/ui/button'
import { Input } from '@/gen/shadcn/ui/input'

// IssueAPIKeyDialogProps defines the component props.
interface IssueAPIKeyDialogProps {
  // onKeyIssued callback when a key is successfully issued, receives the secret key
  onKeyIssued?: (secretKey: string) => void
}

// IssueAPIKeyDialog provides a dialog for creating new API keys.
export const IssueAPIKeyDialog = ({ onKeyIssued }: IssueAPIKeyDialogProps) => {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [issuedKey, setIssuedKey] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const issueMutation = useIssueAPIKey()

  const handleSubmit: FormEventHandler<HTMLFormElement> = (e) => {
    e.preventDefault()
    if (!name.trim()) {
      return
    }

    issueMutation.mutate(
      { name: name.trim(), expiresInDays: 0 },
      {
        onSuccess: (response) => {
          setIssuedKey(response.apiKey)
          onKeyIssued?.(response.apiKey)
        },
      },
    )
  }

  const handleCopy = async () => {
    if (issuedKey) {
      await navigator.clipboard.writeText(issuedKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen)
    if (!newOpen) {
      setName('')
      setIssuedKey(null)
      setCopied(false)
      issueMutation.reset()
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>Create API Key</Button>
      </DialogTrigger>
      <DialogContent>
        {issuedKey ? (
          <>
            <DialogHeader>
              <DialogTitle>API Key Created</DialogTitle>
              <DialogDescription>
                Copy your API key now. You will not be able to see it again.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="rounded-md bg-zinc-900 p-4">
                <code className="break-all text-sm text-zinc-100">
                  {issuedKey}
                </code>
              </div>
              <p className="text-sm text-amber-500">
                Make sure to copy your API key now. You will not be able to see
                it again!
              </p>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => handleOpenChange(false)}>
                Close
              </Button>
              <Button onClick={handleCopy}>
                {copied ? 'Copied!' : 'Copy to Clipboard'}
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>Create API Key</DialogTitle>
              <DialogDescription>
                Create a new API key for accessing the API programmatically.
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={handleSubmit}>
              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <span className="text-sm font-medium">Name</span>
                  <Input
                    placeholder="My API Key"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => handleOpenChange(false)}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  disabled={!name.trim() || issueMutation.isPending}
                >
                  {issueMutation.isPending ? 'Creating...' : 'Create'}
                </Button>
              </DialogFooter>
            </form>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
