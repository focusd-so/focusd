interface Step1Props {
  entered: boolean;
}

export function Step1({ entered }: Step1Props) {
  return (
    <>
      <h1
        className="text-6xl font-bold mb-5 tracking-tight"
        style={{
          background:
            "linear-gradient(100deg, #666 0%, #666 40%, #d0d0d0 50%, #666 60%, #666 100%)",
          backgroundSize: "300% 100%",
          WebkitBackgroundClip: "text",
          WebkitTextFillColor: "transparent",
          backgroundClip: "text",
          paddingBottom: 4,
          animation: entered ? "shimmer 4s linear 4s infinite" : "none",
          opacity: entered ? 1 : 0,
          transform: entered ? "translateY(0)" : "translateY(-6px)",
          transition: "opacity 1.6s ease 0.3s, transform 1.6s ease 0.3s",
        }}
      >
        Claim your focus
      </h1>
      <p
        className="text-xl max-w-md leading-relaxed"
        style={{
          color: "#d4d4d4",
          opacity: entered ? 1 : 0,
          transform: entered ? "translateY(0)" : "translateY(5px)",
          transition: "opacity 1.4s ease 0.8s, transform 1.4s ease 0.8s",
        }}
      >
        More intentional Mac experience starts here.
      </p>
    </>
  );
}
