import { createFileRoute, redirect, useSearch } from '@tanstack/react-router'
import { AlertCircle, Shield } from 'lucide-react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'
import { DeviceApproval } from '@/feature/auth/component/device-approval'

// DeviceSearchParams defines the search params type for this route.
interface DeviceSearchParams {
  code?: string
}

// Route configuration using TanStack Router's createFileRoute.
export const Route = createFileRoute('/auth/device')({
  beforeLoad: ({ location }) => {
    // Read access token from localStorage
    const token = localStorage.getItem('cops_access_token')

    // If token does not exist or has length === 0, redirect to login
    if (!token || token.length === 0) {
      // Build redirect URL with full path and search params
      const redirectUrl = location.pathname + location.searchStr

      // Throw redirect to login page with redirect param
      throw redirect({
        to: '/auth',
        search: { redirect: redirectUrl },
      })
    }
  },
  component: DeviceApprovalPage,
  validateSearch: (search: Record<string, unknown>): DeviceSearchParams => {
    return {
      code: typeof search.code === 'string' ? search.code : undefined,
    }
  },
})

function DeviceApprovalPage() {
  const search = useSearch({ from: '/auth/device' })
  const code = search.code

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <div className="w-full max-w-md px-4">
        {!code ? (
          <Card className="border-zinc-800 bg-zinc-900">
            <CardHeader>
              <div className="flex items-center gap-3">
                <AlertCircle className="h-6 w-6 text-amber-400" />
                <div>
                  <CardTitle className="text-zinc-100">
                    No Device Code
                  </CardTitle>
                  <CardDescription className="text-zinc-500">
                    No device code provided
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <Alert className="border-amber-900/50 bg-amber-950/30">
                <AlertDescription className="text-amber-200">
                  Please use the link from your CLI.
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <div className="rounded-lg border border-cyan-500/20 bg-zinc-900/80 p-2">
                <Shield className="h-5 w-5 text-cyan-400" />
              </div>
              <div>
                <h1 className="text-xl font-bold text-zinc-100">
                  Approve CLI Access
                </h1>
                <p className="text-sm text-zinc-500">
                  Review and approve this device code to sign in
                </p>
              </div>
            </div>
            <DeviceApproval userCode={code} />
          </div>
        )}
      </div>
    </div>
  )
}
