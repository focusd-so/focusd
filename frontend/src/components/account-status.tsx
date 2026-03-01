import { IconFlame, IconAlertCircle, IconStar, IconCrown } from "@tabler/icons-react";
import { Browser } from "@wailsio/runtime";
import { Button } from "@/components/ui/button";
import { useAccountStore } from "@/stores/account-store";
import { DeviceHandshakeResponse_AccountTier } from "../../bindings/github.com/focusd-so/focusd/gen/api/v1/models";
import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

export function AccountStatus() {
  const { checkoutLink, fetchAccountTier, isLoadingAccountTier: isStoreLoading } = useAccountStore();
  const queryClient = useQueryClient();

  const { data: accountTier, isLoading: isQueryLoading } = useQuery({
    queryKey: ['accountTier'],
    queryFn: () => fetchAccountTier()
  });

  // When the backend signals identity has changed (e.g. after checkout),
  // update the query cache directly with the tier from the event payload.
  useEffect(() => {
    const handler = (e: Event) => {
      const tier = (e as CustomEvent).detail;
      if (tier != null) {
        queryClient.setQueryData(['accountTier'], tier);
      }
    };
    window.addEventListener("authctx:updated", handler);
    return () => window.removeEventListener("authctx:updated", handler);
  }, [queryClient]);

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

  // BASIC tier - compact badge
  if (accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_BASIC) {
    return (
      <div className="flex items-center gap-1.5 h-7 px-3 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400">
        <IconStar className="w-3.5 h-3.5 fill-emerald-400/20" />
        <span className="text-xs font-semibold tracking-wide">Basic</span>
      </div>
    );
  }

  // PRO tier - compact badge
  if (accountTier === DeviceHandshakeResponse_AccountTier.DeviceHandshakeResponse_ACCOUNT_TIER_PRO) {
    return (
      <div className="flex items-center gap-1.5 h-7 px-3 rounded-full bg-violet-500/10 border border-violet-500/20 text-violet-400 shadow-[0_0_10px_rgba(139,92,246,0.1)]">
        <IconCrown className="w-3.5 h-3.5 fill-violet-400/20" />
        <span className="text-xs font-semibold tracking-wide uppercase">Pro</span>
      </div>
    );
  }

  // UNSPECIFIED or unknown - compact debug badge
  return <></>;
}
