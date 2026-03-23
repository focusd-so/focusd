package usage

import apiv1 "github.com/focusd-so/focusd/gen/api/v1"

func hasCustomRulesExecutionAccess(tier apiv1.DeviceHandshakeResponse_AccountTier) bool {
	switch tier {
	case apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL,
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PLUS,
		apiv1.DeviceHandshakeResponse_ACCOUNT_TIER_PRO:
		return true
	default:
		return false
	}
}
