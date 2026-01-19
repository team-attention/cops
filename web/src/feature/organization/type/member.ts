// MemberWithDetails represents a member with full user information.
// Used in the members list display.
export interface MemberWithDetails {
  userId: string
  email: string
  name: string
  avatarUrl: string
  role: 'admin' | 'member'
}

// EditOrganizationFormData represents the form state for editing organization.
export interface EditOrganizationFormData {
  name: string
  slug: string
}

// SlugValidationResult represents slug validation state.
export interface SlugValidationResult {
  isValid: boolean
  errorMessage: string | null
}
