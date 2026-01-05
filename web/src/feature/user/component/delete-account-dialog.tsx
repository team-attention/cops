import { useCallback, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { AlertTriangle, Loader2 } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useDeleteAccount } from '../hook/use-delete-account'
import { useAuth } from '@/shared/hook/use-auth'
import { useUserStore } from '@/shared/store/user-store'
import { Button } from '@/gen/shadcn/ui/button'
import { Input } from '@/gen/shadcn/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/gen/shadcn/ui/dialog'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

// DeleteAccountDialogState represents the dialog's internal state.
type DeleteAccountDialogState =
  | { status: 'idle' }
  | { status: 'confirming' }
  | { status: 'error'; message: string }

interface DeleteAccountDialogProps {
  // trigger is the element that opens the dialog when clicked
  trigger: React.ReactNode
}

export const DeleteAccountDialog = ({ trigger }: DeleteAccountDialogProps) => {
  // 1. useState for:
  //    - isOpen: boolean (dialog open state)
  //    - confirmationInput: string (user's typed confirmation)
  //    - state: DeleteAccountDialogState (idle, confirming, error)
  const [isOpen, setIsOpen] = useState(false)
  const [confirmationInput, setConfirmationInput] = useState('')
  const [state, setState] = useState<DeleteAccountDialogState>({
    status: 'idle',
  })

  // 2. Get mutation from useDeleteAccount()
  const mutation = useDeleteAccount()

  // 3. Get logout from useAuth()
  const { logout } = useAuth()

  // 4. Get reset from useUserStore()
  const reset = useUserStore((state) => state.reset)

  // 5. Get navigate from useNavigate()
  const navigate = useNavigate()

  // 6. Create handleConfirmationChange callback:
  //    a. Update confirmationInput state
  //    b. Clear any error state (set to idle)
  const handleConfirmationChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setConfirmationInput(e.target.value)
      setState({ status: 'idle' })
    },
    [],
  )

  // 7. Create handleDelete async callback:
  //    a. Set state to confirming
  //    b. Try:
  //       i. Call mutation.mutateAsync({ confirmationPhrase: confirmationInput })
  //       ii. Call logout() to clear tokens
  //       iii. Call reset() to clear user store
  //       iv. Navigate to '/' (home page)
  //    c. Catch (error):
  //       i. Map error.code to user-friendly message:
  //          - Code.InvalidArgument: "Please type 'DELETE' exactly to confirm"
  //          - Code.Unauthenticated: "Session expired. Please log in again."
  //          - Default: error.message or "An error occurred"
  //       ii. Set state to error with message
  const handleDelete = useCallback(async () => {
    setState({ status: 'confirming' })
    try {
      await mutation.mutateAsync({ confirmationPhrase: confirmationInput })
      logout()
      reset()
      navigate({ to: '/' })
    } catch (error) {
      const connectError = error as { code?: Code; message?: string }
      let errorMessage = 'An error occurred'

      if (connectError.code === Code.InvalidArgument) {
        errorMessage = "Please type 'DELETE' exactly to confirm"
      } else if (connectError.code === Code.Unauthenticated) {
        errorMessage = 'Session expired. Please log in again.'
      } else if (connectError.message) {
        errorMessage = connectError.message
      }

      setState({ status: 'error', message: errorMessage })
    }
  }, [mutation, confirmationInput, logout, reset, navigate])

  // 8. Create isConfirmationValid computed:
  //    a. Return confirmationInput === 'DELETE'
  const isConfirmationValid = confirmationInput === 'DELETE'

  // 9. Create handleOpenChange callback:
  //    a. If closing (open is false), reset state to idle and clear confirmationInput
  //    b. Update isOpen state
  const handleOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setState({ status: 'idle' })
      setConfirmationInput('')
    }
    setIsOpen(open)
  }, [])

  // 10. Return Dialog with:
  //     - open={isOpen}
  //     - onOpenChange={handleOpenChange}
  //     - DialogTrigger with asChild wrapping {trigger}
  //     - DialogContent containing:
  //       a. DialogHeader with warning icon and title "Delete Account"
  //       b. DialogDescription explaining the permanent action and cascade deletion
  //       c. Warning alert with list of what will be deleted:
  //          - All your personal data and authentication accounts
  //          - Organizations where you are the sole member (with projects and sessions)
  //          - Your membership in shared organizations
  //       d. Input field for typing 'DELETE'
  //       e. Error alert (shown when state.status === 'error')
  //       f. DialogFooter with:
  //          - Cancel button (type="button", variant="outline")
  //          - Delete button (variant="destructive", disabled when invalid or confirming)
  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-3">
            <AlertTriangle className="h-6 w-6 text-red-400" />
            <DialogTitle>Delete Account</DialogTitle>
          </div>
          <DialogDescription>
            This action is permanent and cannot be undone. Your account and all
            related data will be permanently deleted.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <Alert className="border-red-900/50 bg-red-950/20">
            <AlertDescription className="text-sm text-red-200">
              <p className="mb-2 font-semibold">This will delete:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>All your personal data and authentication accounts</li>
                <li>
                  Organizations where you are the sole member (including their
                  projects and sessions)
                </li>
                <li>Your membership in shared organizations</li>
              </ul>
            </AlertDescription>
          </Alert>

          <div className="space-y-2">
            <label
              htmlFor="confirmation"
              className="text-sm font-medium text-zinc-300"
            >
              Type <span className="font-mono text-red-400">DELETE</span> to
              confirm:
            </label>
            <Input
              id="confirmation"
              value={confirmationInput}
              onChange={handleConfirmationChange}
              placeholder="DELETE"
              className="font-mono"
            />
          </div>

          {state.status === 'error' && (
            <Alert className="border-red-900/50 bg-red-950/30">
              <AlertDescription className="text-red-200">
                {state.message}
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={state.status === 'confirming'}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={!isConfirmationValid || state.status === 'confirming'}
          >
            {state.status === 'confirming' ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Deleting...
              </>
            ) : (
              'Delete Account'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
