import { useState, useEffect } from 'react';
import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { CheckCircle, XCircle, Loader2 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card';
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert';
import { Button } from '@/gen/shadcn/ui/button';
import { useAuth } from '@/shared/hook/use-auth';
import { useGoogleAuth } from '@/feature/auth/hook/use-google-auth';

// CallbackSearchParams defines search params for OAuth callback
interface CallbackSearchParams {
  code?: string;
  error?: string;
}

// Route configuration
export const Route = createFileRoute('/auth/callback')({
  component: CallbackPage,
  validateSearch: (search: Record<string, unknown>): CallbackSearchParams => {
    return {
      code: typeof search.code === 'string' ? search.code : undefined,
      error: typeof search.error === 'string' ? search.error : undefined,
    };
  },
});

// CallbackState represents the callback processing state
interface CallbackPending {
  status: 'pending';
}

interface CallbackSuccess {
  status: 'success';
}

interface CallbackError {
  status: 'error';
  message: string;
}

type CallbackState = CallbackPending | CallbackSuccess | CallbackError;

// CallbackPage processes OAuth callback and exchanges code for tokens
function CallbackPage() {
  const search = useSearch({ from: '/auth/callback' });
  const navigate = useNavigate();
  const { storeTokens } = useAuth();
  const mutation = useGoogleAuth();
  const [state, setState] = useState<CallbackState>({ status: 'pending' });

  useEffect(() => {
    const processCallback = async () => {
      // If search.error exists, set error state
      if (search.error) {
        setState({
          status: 'error',
          message: search.error,
        });
        return;
      }

      // If search.code does not exist, set error state
      if (!search.code) {
        setState({
          status: 'error',
          message: 'No authorization code received',
        });
        return;
      }

      // Retrieve stored redirect URL from sessionStorage
      const storedRedirect = sessionStorage.getItem('cops_oauth_redirect');

      // Build redirect URI from environment variable
      const redirectUri = import.meta.env.VITE_GOOGLE_OAUTH_REDIRECT_URI;

      try {
        // Call mutation with authorization code and redirect URI
        const response = await mutation.mutateAsync({
          authorizationCode: search.code,
          redirectUri,
        });

        // Extract tokens from response
        if (response.tokens) {
          // Store tokens in localStorage
          storeTokens(response.tokens);

          // Remove redirect from sessionStorage
          sessionStorage.removeItem('cops_oauth_redirect');

          // Set success state
          setState({ status: 'success' });

          // Navigate to stored redirect URL or dashboard
          const targetUrl = storedRedirect || '/dashboard';
          window.location.href = targetUrl;
        } else {
          setState({
            status: 'error',
            message: 'No tokens received from server',
          });
        }
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'An error occurred';
        setState({
          status: 'error',
          message: errorMessage,
        });
      }
    };

    processCallback();
  }, [search.code, search.error, mutation, storeTokens, navigate]);

  if (state.status === 'pending') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950">
        <div className="w-full max-w-md px-4">
          <Card className="border-zinc-800 bg-zinc-900">
            <CardHeader>
              <div className="flex items-center gap-3">
                <Loader2 className="h-6 w-6 animate-spin text-cyan-400" />
                <CardTitle className="text-zinc-100">Completing sign in...</CardTitle>
              </div>
            </CardHeader>
          </Card>
        </div>
      </div>
    );
  }

  if (state.status === 'success') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950">
        <div className="w-full max-w-md px-4">
          <Card className="border-green-900/50 bg-green-950/20">
            <CardHeader>
              <div className="flex items-center gap-3">
                <CheckCircle className="h-6 w-6 text-green-400" />
                <CardTitle className="text-green-100">Signed in successfully!</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <Alert className="border-green-900/50 bg-green-950/30">
                <AlertDescription className="text-green-200">
                  Redirecting...
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <div className="w-full max-w-md px-4">
        <Card className="border-red-900/50 bg-red-950/20">
          <CardHeader>
            <div className="flex items-center gap-3">
              <XCircle className="h-6 w-6 text-red-400" />
              <CardTitle className="text-red-100">Error</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <Alert className="border-red-900/50 bg-red-950/30">
              <AlertDescription className="text-red-200">
                {state.message}
              </AlertDescription>
            </Alert>
            <Button
              onClick={() => navigate({ to: '/auth' })}
              className="w-full bg-cyan-600 text-white hover:bg-cyan-500"
            >
              Try Again
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
