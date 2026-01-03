import { Link, useNavigate } from '@tanstack/react-router'
import { Terminal, UserCircle, Settings, LogOut } from 'lucide-react'
import { Button } from '@/gen/shadcn/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/gen/shadcn/ui/dropdown-menu'
import { useAuth } from '@/shared/hook/use-auth'

// LandingHeader displays authentication-aware navigation at top of landing page.
// Authenticated: Dashboard button + Account dropdown (Settings, Logout)
// Unauthenticated: Login button
export const LandingHeader = () => {
  const { isAuthenticated, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
  }

  return (
    <header className="fixed top-0 left-0 right-0 z-50 h-16 border-b border-zinc-800/50 bg-zinc-950/80 backdrop-blur-xl">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 h-full flex items-center justify-between">
        {/* Logo Section */}
        <div className="flex items-center gap-3">
          <div className="rounded-lg border border-cyan-500/20 bg-zinc-900/80 p-2">
            <Terminal className="h-5 w-5 text-cyan-400" />
          </div>
          <span className="font-mono text-lg font-bold tracking-wider text-zinc-100">
            C-OPS
          </span>
        </div>

        {/* Navigation Section */}
        <div className="flex items-center gap-3">
          {isAuthenticated ? (
            <>
              {/* Dashboard Button */}
              <Button
                asChild
                variant="ghost"
                className="text-zinc-300 hover:text-zinc-100 hover:bg-zinc-800/50"
              >
                <Link to="/dashboard">Dashboard</Link>
              </Button>

              {/* Account Dropdown */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800/50"
                  >
                    <UserCircle className="h-5 w-5" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  align="end"
                  className="w-48 border-zinc-800 bg-zinc-900"
                >
                  <DropdownMenuItem onClick={() => navigate({ to: '/settings' })}>
                    <Settings className="h-4 w-4 mr-2" />
                    Account Settings
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={handleLogout}
                    className="text-red-400 focus:text-red-300 focus:bg-red-500/10"
                  >
                    <LogOut className="h-4 w-4 mr-2" />
                    Logout
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          ) : (
            /* Login Button */
            <Button
              asChild
              variant="ghost"
              className="text-zinc-300 hover:text-zinc-100 hover:bg-zinc-800/50"
            >
              <Link to="/auth">Login</Link>
            </Button>
          )}
        </div>
      </div>
    </header>
  )
}
