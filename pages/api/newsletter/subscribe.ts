const TAGS = {
  MAILING_LIST: "mailing-list",
};

export default async (req, res) => {
  if (req.method !== "POST") {
    return res.status(405).json({ error: "Method not allowed" });
  }
  const { email, tags } = req.body;

  if (!email) {
    return res.status(400).json({ error: "Email is required" });
  }

  // Skip in development so we don't add to the list
  if (process.env.NODE_ENV === "development") {
    console.log("Skipping newsletter subscription in development");
    console.log({ email, tags });
    return res.status(201).json({ error: "" });
  }

  try {
    const LIST_ID = process.env.MAILCHIMP_LIST_ID;
    const API_KEY = process.env.MAILCHIMP_API_KEY;
    const DATACENTER = process.env.MAILCHIMP_API_SERVER;
    const data = {
      email_address: email,
      status: "subscribed",
      tags: [TAGS.MAILING_LIST, ...tags],
    };

    const response = await fetch(
      `https://${DATACENTER}.api.mailchimp.com/3.0/lists/${LIST_ID}/members`,

      {
        body: JSON.stringify(data),
        headers: {
          Authorization: `apikey ${API_KEY}`,
          "Content-Type": "application/json",
        },
        method: "POST",
      }
    );

    if (response.status >= 400) {
      const text = await response.text();
      console.log(text);
      return res.status(400).json({
        error: `There was an error subscribing to the newsletter. Please try again.`,
      });
    }

    return res.status(201).json({ error: "" });
  } catch (error) {
    return res.status(500).json({ error: error.message || error.toString() });
  }
};
