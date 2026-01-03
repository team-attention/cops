import { useEffect } from 'react';
import { createFileRoute, useNavigate, useSearch } from '@tanstack/react-router';
import { Shield } from 'lucide-react';
import { Button } from '@/gen/shadcn/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card';
import { useAuth } from '@/shared/hook/use-auth';

// Constants
const GOOGLE_OAUTH_CLIENT_ID = import.meta.env.VITE_GOOGLE_OAUTH_CLIENT_ID;
const GOOGLE_OAUTH_REDIRECT_URI = import.meta.env.VITE_GOOGLE_OAUTH_REDIRECT_URI;
const GOOGLE_OAUTH_SCOPES = 'openid email profile';
const GOOGLE_OAUTH_AUTHORIZE_URL = 'https://accounts.google.com/o/oauth2/v2/auth';

// AuthSearchParams defines search params for auth route
interface AuthSearchParams {
  redirect?: string;
}

// Route configuration
export const Route = createFileRoute('/auth/')({
  component: AuthPage,
  validateSearch: (search: Record<string, unknown>): AuthSearchParams => {
    return {
      redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
    };
  },
});

// AuthPage displays the auth landing page with Google sign-in button
function AuthPage() {
  const search = useSearch({ from: '/auth/' });
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  useEffect(() => {
    // If user is already authenticated, redirect to appropriate page
    if (isAuthenticated) {
      // If redirect param exists, navigate to redirect URL
      if (search.redirect) {
        navigate({ to: search.redirect });
        return;
      }
      // Otherwise, navigate to dashboard
      navigate({ to: '/dashboard' });
    }
  }, [isAuthenticated, search.redirect, navigate]);

  const handleGoogleSignIn = () => {
    // Store redirect URL in sessionStorage if it exists
    if (search.redirect) {
      sessionStorage.setItem('cops_oauth_redirect', search.redirect);
    }

    // Build Google OAuth URL
    const params = new URLSearchParams({
      response_type: 'code',
      client_id: GOOGLE_OAUTH_CLIENT_ID,
      redirect_uri: GOOGLE_OAUTH_REDIRECT_URI,
      scope: GOOGLE_OAUTH_SCOPES,
      access_type: 'offline',
      prompt: 'consent',
    });

    const oauthUrl = `${GOOGLE_OAUTH_AUTHORIZE_URL}?${params.toString()}`;

    // Redirect browser to Google OAuth URL
    window.location.href = oauthUrl;
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <div className="w-full max-w-md px-4">
        <Card className="border-zinc-800 bg-zinc-900">
          <CardHeader className="text-center">
            <div className="mb-4 flex justify-center">
              <div className="rounded-lg border border-cyan-500/20 bg-zinc-900/80 p-3">
                <Shield className="h-8 w-8 text-cyan-400" />
              </div>
            </div>
            <CardTitle className="text-xl text-zinc-100">
              Sign in to C-Ops
            </CardTitle>
            <CardDescription className="text-zinc-500">
              Continue with your Google account
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={handleGoogleSignIn}
              className="w-full bg-white text-zinc-900 hover:bg-zinc-100"
            >
              Sign in with Google
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
