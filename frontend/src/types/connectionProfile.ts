/**
 * The Connection Profile domain type (internal/connectionprofile),
 * matching internal/connectionprofile/httpapi/dto.go's
 * connectionProfileResponse -- a named, reusable set of connection
 * parameters (protocol, port, timeout, host-key policy, and which
 * Authentication to log in with) that an OLT references to describe how
 * to reach and log in to it (see @/types/olt's connectionProfileId).
 */
export interface ConnectionProfile {
  id: string
  name: string
  protocol: string
  port: number
  authenticationId: string | null
  timeout: string
  hostKeyPolicy: string
  description: string
  createdAt: string
  updatedAt: string
}
