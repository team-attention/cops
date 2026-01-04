import { useEffect } from 'react'
import { useUserStore } from '@/shared/store/user-store'
import { useGetMe } from '@/feature/user/hook/use-get-me'
import { useAuth } from '@/shared/hook/use-auth'

// useUser provides access to user state and handles data synchronization.
// Automatically fetches user data when authenticated and syncs to Zustand.
export const useUser = () => {
  const { isAuthenticated } = useAuth()

  const {
    user,
    organizations,
    selectedOrganizationId,
    isLoading,
    error,
    setUser,
    setOrganizations,
    setLoading,
    setError,
    setSelectedOrganizationId,
    reset,
  } = useUserStore()

  const {
    data,
    isLoading: isQueryLoading,
    isError,
    error: queryError,
    refetch,
  } = useGetMe({ enabled: isAuthenticated })

  // Sync query state to Zustand store
  useEffect(() => {
    setLoading(isQueryLoading)
  }, [isQueryLoading, setLoading])

  useEffect(() => {
    if (isError && queryError) {
      setError(queryError.message)
    } else {
      setError(null)
    }
  }, [isError, queryError, setError])

  useEffect(() => {
    if (data) {
      // Map protobuf User to UserData
      if (data.user) {
        setUser({
          id: data.user.id,
          email: data.user.email,
          name: data.user.name,
          avatarUrl: data.user.avatarUrl,
        })
      }

      // Map protobuf Organizations to OrganizationData[]
      const userId = data.user?.id
      const orgs = data.organizations.map((org) => {
        // Find the current user's membership in this organization
        const membership = org.members.find((m) => m.userId === userId)
        return {
          id: org.id,
          name: org.name,
          slug: org.slug,
          role: (membership?.role || 'member') as 'admin' | 'member',
        }
      })
      setOrganizations(orgs)
    }
  }, [data, setUser, setOrganizations])

  // Get selected organization object
  const selectedOrganization = organizations.find(
    (org) => org.id === selectedOrganizationId,
  )

  return {
    user,
    organizations,
    selectedOrganization,
    selectedOrganizationId,
    isLoading,
    error,
    setSelectedOrganizationId,
    refetch,
    reset,
  }
}
