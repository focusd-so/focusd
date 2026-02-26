import React from "react";

interface OnboardingCardProps {
    icon: React.ReactNode;
    title: string;
    description: string;
    action?: React.ReactNode;
    status?: "pending" | "granted" | "denied";
    className?: string;
    style?: React.CSSProperties;
}

export function OnboardingCard({
    icon,
    title,
    description,
    action,
    status,
    className = "",
    style,
}: OnboardingCardProps) {
    return (
        <div
            className={`group relative flex items-center gap-4 rounded-xl px-4 py-4 transition-all duration-500 ${className}`}
            style={{
                background: "rgba(255, 255, 255, 0.03)",
                border: "1px solid rgba(255, 255, 255, 0.08)",
                backdropFilter: "blur(12px)",
                ...style,
            }}
        >
            {/* Icon Container */}
            <div
                className="flex items-center justify-center w-10 h-10 rounded-lg shrink-0 transition-all duration-500"
                style={{
                    background:
                        status === "granted"
                            ? "rgba(34, 197, 94, 0.1)"
                            : "rgba(255, 255, 255, 0.05)",
                    color: status === "granted" ? "#4ade80" : "inherit",
                    border: "1px solid",
                    borderColor:
                        status === "granted"
                            ? "rgba(34, 197, 94, 0.2)"
                            : "rgba(255, 255, 255, 0.1)",
                }}
            >
                {icon}
            </div>

            {/* Text Content */}
            <div className="flex-1 min-w-0 text-left">
                <h3 className="text-sm font-semibold text-zinc-100 mb-1 leading-none">
                    {title}
                </h3>
                <p className="text-xs text-zinc-400 leading-snug">{description}</p>
            </div>

            {/* Action Area */}
            {action && <div className="shrink-0 ml-2">{action}</div>}
        </div>
    );
}
