import { useCallback, useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useUpdateOrganization } from '../hook/use-update-organization'
import type {
  EditOrganizationFormData,
  SlugValidationResult,
} from '../type/member'
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

// EditOrganizationDialogState represents the dialog's internal state.
type EditOrganizationDialogState =
  | { status: 'idle' }
  | { status: 'submitting' }
  | { status: 'error'; message: string }

interface EditOrganizationDialogProps {
  // organizationId is the ID of the organization to edit
  organizationId: string
  // currentName is the current organization name
  currentName: string
  // currentSlug is the current organization slug
  currentSlug: string
  // trigger is the element that opens the dialog when clicked
  trigger: React.ReactNode
}

// validateSlug validates the slug format.
// Returns validation result with isValid and errorMessage.
const validateSlug = (slug: string): SlugValidationResult => {
  const trimmed = slug.trim().toLowerCase()

  if (trimmed === '') {
    return { isValid: false, errorMessage: 'Slug is required' }
  }

  if (trimmed.length < 3) {
    return {
      isValid: false,
      errorMessage: 'Slug must be at least 3 characters',
    }
  }

  if (trimmed.length > 63) {
    return {
      isValid: false,
      errorMessage: 'Slug must be at most 63 characters',
    }
  }

  const slugPattern = /^[a-z0-9]+(-[a-z0-9]+)*$/
  if (!slugPattern.test(trimmed)) {
    return {
      isValid: false,
      errorMessage:
        'Slug must contain only lowercase letters, numbers, and hyphens',
    }
  }

  return { isValid: true, errorMessage: null }
}

export const EditOrganizationDialog = ({
  organizationId,
  currentName,
  currentSlug,
  trigger,
}: EditOrganizationDialogProps) => {
  const [isOpen, setIsOpen] = useState(false)
  const [formData, setFormData] = useState<EditOrganizationFormData>({
    name: currentName,
    slug: currentSlug,
  })
  const [state, setState] = useState<EditOrganizationDialogState>({
    status: 'idle',
  })
  const [slugValidation, setSlugValidation] = useState<SlugValidationResult>({
    isValid: true,
    errorMessage: null,
  })

  const mutation = useUpdateOrganization()
  const updateOrganization = useUserStore((state) => state.updateOrganization)

  // Reset form when dialog opens with current values
  useEffect(() => {
    if (isOpen) {
      setFormData({
        name: currentName,
        slug: currentSlug,
      })
      setState({ status: 'idle' })
      setSlugValidation({ isValid: true, errorMessage: null })
    }
  }, [isOpen, currentName, currentSlug])

  const handleNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setFormData((prev) => ({ ...prev, name: e.target.value }))
      setState({ status: 'idle' })
    },
    [],
  )

  const handleSlugChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const slug = e.target.value.toLowerCase().trim()
      setFormData((prev) => ({ ...prev, slug }))
      const validation = validateSlug(slug)
      setSlugValidation(validation)
      setState({ status: 'idle' })
    },
    [],
  )

  const handleSubmit = useCallback(async () => {
    const validation = validateSlug(formData.slug)
    if (!validation.isValid) {
      setState({
        status: 'error',
        message: validation.errorMessage || 'Invalid slug',
      })
      return
    }

    setState({ status: 'submitting' })
    try {
      const result = await mutation.mutateAsync({
        organizationId,
        name: formData.name,
        slug: formData.slug,
      })

      // Update zustand store with new organization data
      if (result.organization) {
        updateOrganization(organizationId, {
          name: result.organization.name,
          slug: result.organization.slug,
        })
      }

      setIsOpen(false)
    } catch (error) {
      const connectError = error as { code?: Code; message?: string }
      let errorMessage = 'An error occurred'

      if (connectError.code === Code.PermissionDenied) {
        errorMessage = "You don't have permission to edit this organization"
      } else if (connectError.code === Code.InvalidArgument) {
        errorMessage = connectError.message || 'Invalid input'
      } else if (connectError.message) {
        errorMessage = connectError.message
      }

      setState({ status: 'error', message: errorMessage })
    }
  }, [mutation, organizationId, formData, updateOrganization])

  const isFormValid = formData.name.trim() !== '' && slugValidation.isValid
  const hasChanges =
    formData.name !== currentName || formData.slug !== currentSlug

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Organization</DialogTitle>
          <DialogDescription>
            Update your organization's name and slug. The slug is used in URLs.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <label htmlFor="name" className="text-sm font-medium">
              Organization Name
            </label>
            <Input
              id="name"
              value={formData.name}
              onChange={handleNameChange}
              placeholder="My Organization"
              disabled={state.status === 'submitting'}
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="slug" className="text-sm font-medium">
              Slug
            </label>
            <Input
              id="slug"
              value={formData.slug}
              onChange={handleSlugChange}
              placeholder="my-organization"
              disabled={state.status === 'submitting'}
            />
            {slugValidation.errorMessage && (
              <p className="text-sm text-red-500">
                {slugValidation.errorMessage}
              </p>
            )}
          </div>

          {state.status === 'error' && (
            <Alert className="border-red-900/50 bg-red-950/20">
              <AlertDescription className="text-sm text-red-200">
                {state.message}
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => setIsOpen(false)}
            disabled={state.status === 'submitting'}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={
              !isFormValid || !hasChanges || state.status === 'submitting'
            }
          >
            {state.status === 'submitting' && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
