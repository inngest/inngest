import { useRef, useState } from "react";
import Header from "src/shared/Header";
import Container from "src/shared/layout/Container";
import Footer from "src/shared/Footer";

export default function LaunchWeek() {
  return (
    <div className="home font-sans bg-slate-1000 bg-[url(/assets/launch-week/background-image.png)] bg-cover">
      <Header />
      <Container className="py-8">
        <div className="my-12 tracking-tight flex items-center justify-center">
          <div className="py-12 md:py-24 rounded-md">
            <h1 className="font-bold text-5xl md:text-7xl leading-tight md:leading-tight text-white text-center">
              inngest <br />
              <span className="bg-clip-text text-transparent bg-gradient-to-br bg-gradient-to-r from-[#5EEAD4] via-[#A7F3D0] to-[#FDE68A]">
                Launch Week
              </span>
            </h1>
            <div className="mt-5 flex items-center justify-center">
              <span
                className="py-2 px-8 uppercase text-white font-extrabold text-lg md:text-xl border-2 border-transparent rounded-full"
                style={{
                  background: `linear-gradient(#292e23, #292e23) padding-box,
                              linear-gradient(to right, #5EEAD4, #A7F3D0, #FDE68A) border-box`,
                }}
              >
                January 22-25 2024
              </span>
            </div>
            <p className="my-12 text-slate-200 text-lg md:text-xl">
              A week of updates from Inngest starting{" "}
              <span className="font-bold bg-clip-text text-transparent bg-gradient-to-br bg-gradient-to-r from-[#5EEAD4] via-[#A7F3D0] to-[#FDE68A]">
                January 22nd, 2024
              </span>
            </p>

            <NewsletterSignup tags={["launch-week-jan-2023"]} />
          </div>
        </div>
      </Container>

      <Footer disableCta={true} />
    </div>
  );
}

function NewsletterSignup({ tags = [] }: { tags: string[] }) {
  const inputRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<{
    error: string;
    result: boolean | null;
  }>({
    error: "",
    result: null,
  });

  const subscribeUser = async (e) => {
    e.preventDefault();
    setLoading(true);
    const res = await fetch("/api/newsletter/subscribe", {
      body: JSON.stringify({
        email: inputRef.current.value,
        tags,
      }),
      headers: {
        "Content-Type": "application/json",
      },
      method: "POST",
    });
    setLoading(false);
    if (res.status === 201) {
      alert("Success! 🎉 You are now subscribed to the newsletter.");
      inputRef.current.value = "";
    }
  };

  return (
    <form onSubmit={subscribeUser}>
      <p className="mb-2 text-white text-sm">Get notified:</p>

      <div className="flex flex-row flex-wrap gap-4">
        <input
          className="w-72 flex-grow border border-slate-400 rounded-md px-4 py-2 text-white bg-transparent focus:outline-none focus:ring-1 focus:ring-[#A7F3D0] focus:border-transparent"
          type="email"
          id="email-input"
          name="email"
          placeholder="Enter your email address"
          ref={inputRef}
          required
          autoCapitalize="off"
          autoCorrect="off"
        />
        <button
          type="submit"
          name="register"
          disabled={loading || response.result === true}
          className="whitespace-nowrap button group inline-flex items-center justify-center gap-0.5 rounded-md font-medium tracking-tight transition-all text-slate-950 placeholder:text-slate-300 bg-gradient-to-br bg-gradient-to-r from-[#5EEAD4] via-[#A7F3D0] to-[#FDE68A] text-sm px-3 py-2"
        >
          Register
        </button>
      </div>
    </form>
  );
}
