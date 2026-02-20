import {
  IconSettings,
  IconShield,
  IconLayoutDashboard,
  IconClock,
  IconTrendingUp,
} from "@tabler/icons-react";
import { Link, useMatchRoute } from "@tanstack/react-router";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

interface MenuItem {
  title: string;
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  children?: { title: string; to: string }[];
}

const applicationItems: MenuItem[] = [
  {
    title: "Smart Blocking",
    to: "/activity",
    icon: IconShield,
  },
];

const insightItems: MenuItem[] = [
  { title: "Overview", to: "/screen-time", icon: IconLayoutDashboard },
  { title: "Screen Time", to: "/screen-time/screentime", icon: IconClock },
  { title: "Trends", to: "/screen-time/trends", icon: IconTrendingUp },
];

export function AppSidebar() {
  const matchRoute = useMatchRoute();

  return (
    <Sidebar variant="floating">
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Application</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {applicationItems.map((item) => {
                const isActive = !!matchRoute({ to: item.to, fuzzy: true });
                return (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild isActive={isActive}>
                      <Link to={item.to}>
                        <item.icon className="w-4 h-4" />
                        <span>{item.title}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

{/* <SidebarGroup>
          <SidebarGroupLabel>Insights</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {insightItems.map((item) => {
                const isActive = !!matchRoute({ to: item.to, fuzzy: false });
                return (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild isActive={isActive}>
                      <Link to={item.to}>
                        <item.icon className="w-4 h-4" />
                        <span>{item.title}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup> */}
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              isActive={!!matchRoute({ to: "/settings" })}
            >
              <Link to="/settings">
                <IconSettings className="w-4 h-4" />
                <span>Settings</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}
