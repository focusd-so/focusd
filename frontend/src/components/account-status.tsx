import { IconSparkles, IconFlame } from "@tabler/icons-react";
import { Browser } from "@wailsio/runtime";
import { Button } from "@/components/ui/button";
import { useAccountStore } from "@/stores/account-store";
import { DeviceHandshakeResponse_AccountTier } from "../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";

export function AccountStatus() {
  const { accountTier, checkoutLink, isLoading } = useAccountStore();

  if (isLoading) {
    return null;
  }

  const handleUpgrade = () => {
    if (checkoutLink) {
      Browser.OpenURL(checkoutLink);
    }
  };

  // FREE tier - show upgrade button
  if (accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_FREE) {
    return (
      <Button
        size="sm"
        className="h-7 bg-emerald-600 hover:bg-emerald-700 text-white text-xs"
        onClick={handleUpgrade}
      >
        <IconSparkles className="w-3.5 h-3.5 mr-1" />
        Upgrade
      </Button>
    );
  }

  // TRIAL tier - single compact button with trial info + upgrade CTA
  if (accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL) {
    return (
      <Button
        variant="outline"
        size="sm"
        className="h-7 border-amber-500/30 bg-amber-500/5 text-amber-500 hover:bg-amber-500/10 hover:text-amber-400 text-xs gap-1.5"
        onClick={handleUpgrade}
      >
        <IconFlame className="w-3.5 h-3.5" />
        <span>7 days trial</span>
        <span className="text-amber-500/40">|</span>
        <span className="font-semibold">Upgrade</span>
      </Button>
    );
  }

  // // BASIC tier - compact badge
  // if (accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_BASIC) {
  //   return (
  //     <div className="flex items-center gap-1 h-7 px-2.5 rounded-full bg-emerald-500/10 border border-emerald-500/20">
  //       <IconCrown className="w-3.5 h-3.5 text-emerald-500" />
  //       <span className="text-xs font-medium text-emerald-500">Basic</span>
  //     </div>
  //   );
  // }

  // // PRO tier - compact badge
  // if (accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_PRO) {
  //   return (
  //     <div className="flex items-center gap-1 h-7 px-2.5 rounded-full bg-violet-500/10 border border-violet-500/20">
  //       <IconCrown className="w-3.5 h-3.5 text-violet-500" />
  //       <span className="text-xs font-medium text-violet-500">Pro</span>
  //     </div>
  //   );
  // }

  // UNSPECIFIED or unknown - compact debug badge
  return <></>;
}
