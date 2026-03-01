import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { IconCreditCard } from "@tabler/icons-react";
import { Browser } from "@wailsio/runtime";
import { CustomerPortal } from "../../../bindings/github.com/focusd-so/focusd/internal/identity/service";

export function AccountSettings() {
    const handleManageSubscription = async () => {
        try {
            const url = await CustomerPortal();
            if (url) {
                Browser.OpenURL(url);
            }
        } catch (error) {
            console.error("Failed to fetch customer portal:", error);
        }
    };

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle>Subscription</CardTitle>
                    <CardDescription>
                        Manage your billing and subscription via our secure portal.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="flex items-center justify-between">
                        <div className="space-y-0.5">
                            <div className="text-sm font-medium">Customer Portal</div>
                            <div className="text-sm text-muted-foreground">
                                View invoices, change your plan, or update payment methods.
                            </div>
                        </div>
                        <Button variant="outline" size="sm" onClick={handleManageSubscription}>
                            <IconCreditCard className="w-4 h-4 mr-2" />
                            Manage Subscription
                        </Button>
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
