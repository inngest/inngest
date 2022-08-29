export async function run() {
  return {
    status: 200,
    body: {
      simple: process.env.SIMPLE,
      quoted: process.env.QUOTED,
      quotedEscapes: process.env.QUOTED_ESCAPES,
      certificate: process.env.CERTIFICATE,
      json: JSON.parse(process.env.JSON),
    },
  };
}
