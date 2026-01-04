import { useState, useCallback, useMemo } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Building2, Loader2 } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useCreateOrganization } from '../hook/use-create-organization'
import { useUserStore } from '@/shared/store/user-store'
import { Button } from '@/gen/shadcn/ui/button'
import { Input } from '@/gen/shadcn/ui/input'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

// OrganizationFormState represents the form's internal state.
type OrganizationFormState =
  | { status: 'idle' }
  | { status: 'submitting' }
  | { status: 'error'; message: string }

// slugRegex validates slug format (lowercase alphanumeric, hyphens, underscores)
const slugRegex = /^[a-z0-9-_]*$/

export const OrganizationForm = () => {
  // 1. useState for:
  //    - name: string (initial: '')
  //    - slug: string (initial: '')
  //    - state: OrganizationFormState (initial: { status: 'idle' })
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [state, setState] = useState<OrganizationFormState>({ status: 'idle' })

  // 2. Get mutation from useCreateOrganization()
  const mutation = useCreateOrganization()

  // 3. Get addOrganization from useUserStore((s) => s.addOrganization)
  const addOrganization = useUserStore((s) => s.addOrganization)

  // 4. Get navigate from useNavigate()
  const navigate = useNavigate()

  // 5. Create handleNameChange callback using useCallback:
  //    a. Get value from e.target.value
  //    b. Update name state with value
  //    c. If state.status === 'error', set state to { status: 'idle' }
  const handleNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value
      setName(value)
      setState((prev) => (prev.status === 'error' ? { status: 'idle' } : prev))
    },
    [],
  )

  // 6. Create handleSlugChange callback using useCallback:
  //    a. Get value from e.target.value
  //    b. Convert value to lowercase
  //    c. If lowercase value matches slugRegex OR value is empty string:
  //       i. Update slug state with lowercase value
  //    d. If state.status === 'error', set state to { status: 'idle' }
  const handleSlugChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value
      const lowercase = value.toLowerCase()
      if (slugRegex.test(lowercase) || value === '') {
        setSlug(lowercase)
      }
      setState((prev) => (prev.status === 'error' ? { status: 'idle' } : prev))
    },
    [],
  )

  // 7. Create isFormValid computed using useMemo:
  //    a. Return true if ALL conditions met:
  //       i. name.trim().length > 0
  //       ii. slug.length > 0
  //       iii. slugRegex.test(slug)
  const isFormValid = useMemo(() => {
    return name.trim().length > 0 && slug.length > 0 && slugRegex.test(slug)
  }, [name, slug])

  // 8. Create handleSubmit async callback using useCallback:
  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      //    a. Call e.preventDefault()
      e.preventDefault()
      //    b. If !isFormValid, return early
      if (!isFormValid) {
        return
      }
      //    c. Set state to { status: 'submitting' }
      setState({ status: 'submitting' })
      //    d. Try block:
      try {
        //       i. Call response = await mutation.mutateAsync({ name: name.trim(), slug })
        const response = await mutation.mutateAsync({
          name: name.trim(),
          slug,
        })
        //       ii. If response.organization exists:
        if (response.organization) {
          //           - Create organizationData object with:
          //             * id: response.organization.id
          //             * name: response.organization.name
          //             * role: 'admin' as const
          const organizationData = {
            id: response.organization.id,
            name: response.organization.name,
            slug: response.organization.slug,
            role: 'admin' as const,
          }
          //           - Call addOrganization(organizationData)
          addOrganization(organizationData)
          //           - Call navigate({ to: '/dashboard' })
          navigate({ to: '/dashboard' })
        } else {
          //       iii. Else:
          //           - Set state to { status: 'error', message: 'Failed to create organization' }
          setState({
            status: 'error',
            message: 'Failed to create organization',
          })
        }
      } catch (error) {
        //    e. Catch block (error):
        //       i. Cast error as { code?: Code; message?: string }
        const connectError = error as { code?: Code; message?: string }
        //       ii. Initialize errorMessage = 'An error occurred'
        let errorMessage = 'An error occurred'
        //       iii. Map error.code to message:
        //            - Code.InvalidArgument: 'Please check your input and try again'
        //            - Code.AlreadyExists: 'This slug is already taken. Please choose another.'
        //            - Code.Unauthenticated: 'Session expired. Please log in again.'
        //            - else if error.message exists: use error.message
        if (connectError.code === Code.InvalidArgument) {
          errorMessage = 'Please check your input and try again'
        } else if (connectError.code === Code.AlreadyExists) {
          errorMessage = 'This slug is already taken. Please choose another.'
        } else if (connectError.code === Code.Unauthenticated) {
          errorMessage = 'Session expired. Please log in again.'
        } else if (connectError.message) {
          errorMessage = connectError.message
        }
        //       iv. Set state to { status: 'error', message: errorMessage }
        setState({ status: 'error', message: errorMessage })
      }
    },
    [isFormValid, mutation, name, slug, addOrganization, navigate],
  )

  // 9. Return JSX:
  return (
    <Card className="w-full max-w-md border-zinc-800 bg-zinc-900">
      <CardHeader>
        <div className="flex items-center gap-3">
          <div className="rounded-lg border border-cyan-500/20 bg-cyan-500/10 p-2">
            <Building2 className="h-5 w-5 text-cyan-400" />
          </div>
          <div>
            <CardTitle className="text-zinc-100">Create Organization</CardTitle>
            <CardDescription className="text-zinc-500">
              Create your first organization to get started
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Name field */}
          <div className="space-y-2">
            <label
              htmlFor="name"
              className="text-sm font-medium text-zinc-300"
            >
              Organization Name
            </label>
            <Input
              id="name"
              value={name}
              onChange={handleNameChange}
              placeholder="My Organization"
              className="bg-zinc-800/50"
            />
          </div>

          {/* Slug field */}
          <div className="space-y-2">
            <label
              htmlFor="slug"
              className="text-sm font-medium text-zinc-300"
            >
              URL Slug
            </label>
            <Input
              id="slug"
              value={slug}
              onChange={handleSlugChange}
              placeholder="my-organization"
              className="bg-zinc-800/50 font-mono"
            />
            <p className="text-xs text-zinc-600">
              Lowercase letters, numbers, hyphens, and underscores only
            </p>
          </div>

          {/* Error alert */}
          {state.status === 'error' && (
            <Alert className="border-red-900/50 bg-red-950/30">
              <AlertDescription className="text-red-200">
                {state.message}
              </AlertDescription>
            </Alert>
          )}

          {/* Submit button */}
          <Button
            type="submit"
            disabled={!isFormValid || state.status === 'submitting'}
            className="w-full bg-cyan-600 text-white hover:bg-cyan-500"
          >
            {state.status === 'submitting' ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating...
              </>
            ) : (
              'Create Organization'
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
