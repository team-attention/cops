import { useState } from 'react'
import { CheckCircle, Loader2, XCircle } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useApproveDevice } from '../hook/use-approve-device'
import type { DeviceApprovalState } from '../type/device-code'
import { Button } from '@/gen/shadcn/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

interface DeviceApprovalProps {
  userCode: string
}

type DeviceApprovalErrorCode =
  | 'NOT_FOUND'
  | 'EXPIRED'
  | 'ALREADY_APPROVED'
  | 'UNKNOWN'

export const DeviceApproval = ({ userCode }: DeviceApprovalProps) => {
  const [state, setState] = useState<DeviceApprovalState>({ status: 'pending' })
  const mutation = useApproveDevice()

  const handleApprove = async () => {
    try {
      await mutation.mutateAsync({ userCode })
      setState({
        status: 'success',
        message: 'Device approved successfully!',
      })
    } catch (error) {
      const connectError = error as { code?: Code; message: string }
      const errorCode = connectError.code
      let mappedCode: DeviceApprovalErrorCode = 'UNKNOWN'

      if (errorCode === Code.NotFound) {
        mappedCode = 'NOT_FOUND'
      } else if (errorCode === Code.DeadlineExceeded) {
        mappedCode = 'EXPIRED'
      } else if (errorCode === Code.AlreadyExists) {
        mappedCode = 'ALREADY_APPROVED'
      }
      // Note: UNAUTHORIZED case removed - handled by route guard

      setState({
        status: 'error',
        errorCode: mappedCode,
        message: connectError.message || 'An error occurred',
      })
    }
  }

  if (state.status === 'success') {
    return (
      <Card className="border-green-900/50 bg-green-950/20">
        <CardHeader>
          <div className="flex items-center gap-3">
            <CheckCircle className="h-6 w-6 text-green-400" />
            <CardTitle className="text-green-100">Success</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert className="border-green-900/50 bg-green-950/30">
            <AlertDescription className="text-green-200">
              {state.message}
            </AlertDescription>
          </Alert>
          <p className="text-sm text-zinc-400">
            You can return to your terminal.
          </p>
        </CardContent>
      </Card>
    )
  }

  if (state.status === 'error') {
    const errorMessages: Record<typeof state.errorCode, string> = {
      NOT_FOUND: 'Device code not found. It may have expired.',
      EXPIRED: 'This device code has expired. Please generate a new one.',
      ALREADY_APPROVED: 'This device code has already been approved.',
      UNKNOWN: state.message,
    }

    return (
      <Card className="border-red-900/50 bg-red-950/20">
        <CardHeader>
          <div className="flex items-center gap-3">
            <XCircle className="h-6 w-6 text-red-400" />
            <CardTitle className="text-red-100">Error</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          <Alert className="border-red-900/50 bg-red-950/30">
            <AlertDescription className="text-red-200">
              {errorMessages[state.errorCode]}
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="border-zinc-800 bg-zinc-900">
      <CardHeader>
        <div className="flex items-center gap-3">
          <img src="/logo192.png" alt="C-Ops" className="h-8 w-8" />
          <div>
            <CardTitle className="text-zinc-100">Device Code</CardTitle>
            <CardDescription className="text-zinc-500">
              Approve this code to sign in from your CLI
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="rounded-lg border border-zinc-800 bg-zinc-950/50 p-6 text-center">
          <p className="font-mono text-4xl font-bold tracking-[0.3em] text-cyan-400">
            {userCode}
          </p>
        </div>

        <Button
          onClick={handleApprove}
          disabled={mutation.isPending}
          className="w-full bg-cyan-600 text-white hover:bg-cyan-500 disabled:opacity-50"
        >
          {mutation.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Approving...
            </>
          ) : (
            'Approve Device'
          )}
        </Button>
      </CardContent>
    </Card>
  )
}
