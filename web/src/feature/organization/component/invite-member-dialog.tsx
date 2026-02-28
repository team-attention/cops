import { useCallback, useState } from 'react'
import { Loader2, Mail, UserPlus } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useCreateInvitation } from '../hook/use-create-invitation'
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

// InviteMemberDialogState represents the dialog's internal state.
type InviteMemberDialogState =
  | { status: 'idle' }
  | { status: 'submitting' }
  | { status: 'success' }
  | { status: 'error'; message: string }

interface InviteMemberDialogProps {
  // organizationId is the ID of the organization to invite to
  organizationId: string
  // trigger is the element that opens the dialog when clicked
  trigger: React.ReactNode
  // onSuccess is called when invitation is created successfully
  onSuccess?: () => void
}

// validateEmail validates the email format.
const validateEmail = (email: string): boolean => {
  const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailPattern.test(email.trim())
}

export const InviteMemberDialog = ({
  organizationId,
  trigger,
  onSuccess,
}: InviteMemberDialogProps) => {
  const [isOpen, setIsOpen] = useState(false)
  const [email, setEmail] = useState('')
  const [state, setState] = useState<InviteMemberDialogState>({
    status: 'idle',
  })

  const mutation = useCreateInvitation()

  const handleOpenChange = useCallback((open: boolean) => {
    setIsOpen(open)
    if (!open) {
      // Reset form when dialog closes
      setEmail('')
      setState({ status: 'idle' })
    }
  }, [])

  const handleEmailChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setEmail(e.target.value)
      if (state.status === 'error') {
        setState({ status: 'idle' })
      }
    },
    [state.status],
  )

  const handleSubmit = useCallback(async () => {
    const trimmedEmail = email.trim()

    if (!trimmedEmail) {
      setState({ status: 'error', message: 'Email is required' })
      return
    }

    if (!validateEmail(trimmedEmail)) {
      setState({
        status: 'error',
        message: 'Please enter a valid email address',
      })
      return
    }

    setState({ status: 'submitting' })
    try {
      await mutation.mutateAsync({
        organizationId,
        email: trimmedEmail,
      })

      setState({ status: 'success' })

      // Call onSuccess callback
      onSuccess?.()

      // Close dialog after short delay to show success state
      setTimeout(() => {
        handleOpenChange(false)
      }, 1500)
    } catch (error) {
      const connectError = error as { code?: Code; message?: string }
      let errorMessage = 'An error occurred'

      if (connectError.code === Code.PermissionDenied) {
        errorMessage = 'You must be an admin to invite members'
      } else if (connectError.code === Code.AlreadyExists) {
        errorMessage = connectError.message?.includes('already a member')
          ? 'This user is already a member of the organization'
          : 'An invitation has already been sent to this email'
      } else if (connectError.code === Code.InvalidArgument) {
        errorMessage = connectError.message || 'Invalid input'
      } else if (connectError.message) {
        errorMessage = connectError.message
      }

      setState({ status: 'error', message: errorMessage })
    }
  }, [mutation, organizationId, email, onSuccess, handleOpenChange])

  const isEmailValid = email.trim() !== '' && validateEmail(email)

  return (
    <Dialog open={isOpen} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-3">
            <UserPlus className="h-6 w-6 text-violet-400" />
            <DialogTitle>Invite Member</DialogTitle>
          </div>
          <DialogDescription>
            Send an invitation email to add a new member to your organization.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <label htmlFor="email" className="text-sm font-medium">
              Email Address
            </label>
            <div className="relative">
              <Mail className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-500" />
              <Input
                id="email"
                type="email"
                value={email}
                onChange={handleEmailChange}
                placeholder="colleague@example.com"
                disabled={
                  state.status === 'submitting' || state.status === 'success'
                }
                className="pl-10"
              />
            </div>
          </div>

          {state.status === 'error' && (
            <Alert className="border-red-900/50 bg-red-950/20">
              <AlertDescription className="text-sm text-red-200">
                {state.message}
              </AlertDescription>
            </Alert>
          )}

          {state.status === 'success' && (
            <Alert className="border-green-900/50 bg-green-950/20">
              <AlertDescription className="text-sm text-green-200">
                Invitation sent successfully!
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={state.status === 'submitting'}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={
              !isEmailValid ||
              state.status === 'submitting' ||
              state.status === 'success'
            }
          >
            {state.status === 'submitting' && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            <Mail className="mr-2 h-4 w-4" />
            Send Invitation
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
