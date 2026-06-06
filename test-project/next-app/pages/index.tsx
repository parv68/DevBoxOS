import Head from "next/head";

export default function Home() {
  return (
    <>
      <Head>
        <title>DevBoxOS Next.js App</title>
      </Head>
      <main style={{ padding: "2rem", fontFamily: "sans-serif" }}>
        <h1>DevBoxOS Next.js App</h1>
        <p>Running without Docker via host process runtime.</p>
      </main>
    </>
  );
}
