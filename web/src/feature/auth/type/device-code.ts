// DeviceApprovalPending represents the initial state before approval attempt.
interface DeviceApprovalPending {
  status: 'pending';
}

// DeviceApprovalSuccess represents successful device approval.
interface DeviceApprovalSuccess {
  status: 'success';
  message: string;
}

// DeviceApprovalError represents an error during device approval.
interface DeviceApprovalError {
  status: 'error';
  errorCode: DeviceApprovalErrorCode;
  message: string;
}

// DeviceApprovalErrorCode enumerates possible error conditions.
type DeviceApprovalErrorCode =
  | 'NOT_FOUND'
  | 'EXPIRED'
  | 'ALREADY_APPROVED'
  | 'UNKNOWN';

// DeviceApprovalState is a discriminated union representing all possible states.
export type DeviceApprovalState =
  | DeviceApprovalPending
  | DeviceApprovalSuccess
  | DeviceApprovalError;
