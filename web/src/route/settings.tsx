import { createFileRoute, redirect } from '@tanstack/react-router'
import { Settings, Trash2 } from 'lucide-react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Button } from '@/gen/shadcn/ui/button'
import { DeleteAccountDialog } from '@/feature/user/component/delete-account-dialog'
import { OrganizationSettingsSection } from '@/feature/organization/component/organization-settings-section'
import { APIKeySettings } from '@/feature/api-key/component/api-key-settings'
import { useAuthStore } from '@/shared/store/auth-store'
import { APP_VERSION } from '@/shared/config/version'

export const Route = createFileRoute('/settings')({
  beforeLoad: ({ location }) => {
    // Check authentication status
    const { isAuthenticated } = useAuthStore.getState()
    if (!isAuthenticated) {
      throw redirect({ to: '/auth', search: { redirect: location.href } })
    }
  },
  component: SettingsPage,
})

// SettingsPage displays account settings including the danger zone.
function SettingsPage() {
  return (
    <div className="relative">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8 flex items-center gap-4">
          <div className="relative">
            <div className="absolute inset-0 animate-pulse rounded-xl bg-violet-500/20 blur-xl" />
            <div className="relative rounded-xl border border-violet-500/20 bg-zinc-900/80 p-3 backdrop-blur-sm">
              <Settings className="h-6 w-6 text-violet-400" />
            </div>
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-zinc-100">
              Account Settings
            </h1>
            <p className="mt-0.5 font-mono text-xs text-zinc-600">
              Manage your account
            </p>
          </div>
        </div>

        {/* Organization Settings */}
        <OrganizationSettingsSection />

        {/* Spacer */}
        <div className="h-8" />

        {/* API Key Settings */}
        <APIKeySettings />

        {/* Spacer */}
        <div className="h-8" />

        {/* Danger Zone */}
        <Card className="border-red-900/50 bg-red-950/10">
          <CardHeader>
            <div className="flex items-center gap-3">
              <Trash2 className="h-6 w-6 text-red-400" />
              <div>
                <CardTitle className="text-red-100">Danger Zone</CardTitle>
                <CardDescription className="text-red-200/60">
                  Irreversible and destructive actions
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Delete Account Section */}
            <div className="space-y-3">
              <div>
                <h3 className="text-sm font-semibold text-red-100">
                  Delete Account
                </h3>
                <p className="mt-1 text-sm text-red-200/70">
                  Permanently delete your account and all associated data. This
                  action cannot be undone.
                </p>
              </div>
              <DeleteAccountDialog
                trigger={
                  <Button variant="destructive" size="sm">
                    Delete Account
                  </Button>
                }
              />
            </div>
          </CardContent>
        </Card>

        {/* Footer */}
        <div className="mt-12 flex items-center justify-center gap-2 text-zinc-700">
          <div className="h-px flex-1 bg-gradient-to-r from-transparent to-zinc-800" />
          <span className="font-mono text-[10px] uppercase tracking-widest">
            C-Ops v{APP_VERSION}
          </span>
          <div className="h-px flex-1 bg-gradient-to-l from-transparent to-zinc-800" />
        </div>
      </div>
    </div>
  )
}
