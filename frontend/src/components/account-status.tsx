import { IconFlame, IconAlertCircle } from "@tabler/icons-react";
import { Browser } from "@wailsio/runtime";
import { Button } from "@/components/ui/button";
import { useAccountStore } from "@/stores/account-store";
import { DeviceHandshakeResponse_AccountTier } from "../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";
import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";

export function AccountStatus() {
  const { checkoutLink, fetchAccountTier, isLoadingAccountTier: isStoreLoading } = useAccountStore();

  const { data: accountTier, isLoading: isQueryLoading, refetch } = useQuery({
    queryKey: ['accountTier'],
    queryFn: () => fetchAccountTier()
  });

  // Re-fetch when the backend signals identity has changed (e.g. after checkout)
  useEffect(() => {
    const handler = () => refetch();
    window.addEventListener("authctx:updated", handler);
    return () => window.removeEventListener("authctx:updated", handler);
  }, [refetch]);

  useEffect(() => {
    console.log("accountTier", accountTier)
  }, [accountTier])

  const isLoading = isStoreLoading || isQueryLoading;

  if (isLoading) {
    return null;
  }

  const handleUpgrade = () => {
    if (checkoutLink) {
      Browser.OpenURL(checkoutLink);
    }
  };

  // FREE tier - show urgent upgrade prompt
  if (accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_FREE) {
    return (
      <Button
        variant="outline"
        size="sm"
        className="h-7 border-amber-500/40 bg-amber-500/10 text-amber-400 hover:bg-amber-500/20 hover:text-amber-300 text-xs gap-1.5"
        onClick={handleUpgrade}
      >
        <IconAlertCircle className="w-3.5 h-3.5" />
        <span>Free Plan</span>
        <span className="text-amber-500/40">·</span>
        <span className="font-semibold">Upgrade Now</span>
      </Button>
    );
  }

  // TRIAL tier - urgent "trial ended" CTA
  if (accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_TRIAL) {
    return (
      <Button
        variant="outline"
        size="sm"
        className="h-7 border-amber-500/40 bg-amber-500/10 text-amber-400 hover:bg-amber-500/20 hover:text-amber-300 text-xs gap-1.5"
        onClick={handleUpgrade}
      >
        <IconFlame className="w-3.5 h-3.5" />
        <span>Trial Ended</span>
        <span className="text-amber-500/40">·</span>
        <span className="font-semibold">Upgrade Now</span>
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
