interface OnboardingHeaderProps {
    title: string;
    subtitle: string;
    entered?: boolean;
}

export function OnboardingHeader({ title, subtitle, entered = true }: OnboardingHeaderProps) {
    return (
        <div className="text-center mb-8">
            <h1
                className="text-6xl font-bold mb-5 tracking-tight whitespace-nowrap"
                style={{
                    background: "linear-gradient(to bottom, #FFFFFF 0%, #a3a3a3 100%)",
                    WebkitBackgroundClip: "text",
                    WebkitTextFillColor: "transparent",
                    backgroundClip: "text",
                    paddingBottom: 4,
                    opacity: entered ? 1 : 0,
                    transform: entered ? "translateY(0)" : "translateY(-6px)",
                    transition: "opacity 0.8s ease 0.1s, transform 0.8s ease 0.1s",
                }}
            >
                {title}
            </h1>
            <p
                className="text-xl max-w-2xl mx-auto leading-relaxed"
                style={{
                    color: "#d4d4d4",
                    opacity: entered ? 1 : 0,
                    transform: entered ? "translateY(0)" : "translateY(5px)",
                    transition: "opacity 0.7s ease 0.25s, transform 0.7s ease 0.25s",
                }}
            >
                {subtitle}
            </p>
        </div>
    );
}
