import crypto from "crypto";

const TAGS = {
  MAILING_LIST: "mailing-list",
};
const LIST_ID = process.env.MAILCHIMP_LIST_ID;
const API_KEY = process.env.MAILCHIMP_API_KEY;
const DATACENTER = process.env.MAILCHIMP_API_SERVER;

class MemberExistsError extends Error {}

export default async (req, res) => {
  if (req.method !== "POST") {
    return res.status(405).json({ error: "Method not allowed" });
  }
  const { email, tags = [] }: { email: string | null; tags: string[] | null } =
    req.body;

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
    // Try to add the member to our list
    try {
      const newMemberRes = await addMember(email, [TAGS.MAILING_LIST, ...tags]);
      if (newMemberRes.status >= 400) {
        return res.status(400).json({
          error: `There was an error subscribing to the newsletter. Please try again.`,
        });
      }
    } catch (err) {
      // This error throws if the user is already on the list
      if (err instanceof MemberExistsError) {
        // Update the user's tags
        const updateMemberRes = await addMemberTags(email, tags);
        console.log("res?", JSON.stringify(updateMemberRes, null, 2));
        return res.status(201).json({ error: "" });
      }
      // If it's not an existing member error, throw it
      throw err;
    }

    return res.status(201).json({ error: "" });
  } catch (error) {
    return res.status(500).json({ error: error.message || error.toString() });
  }
};

async function addMember(email: string, tags: string[]) {
  const data = {
    email_address: email,
    status: "subscribed",
    tags,
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
    const text = await response.json();
    if (text.title === "Member Exists") {
      throw new MemberExistsError("Member exists");
    }
  }
  return response;
}

async function addMemberTags(email: string, tags: string[]) {
  const lowercaseEmail = email.toLowerCase();
  const subscriberHash = crypto
    .createHash("md5")
    .update(lowercaseEmail)
    .digest("hex");
  const data = {
    tags: tags.map((t) => ({ name: t, status: "active" })),
  };
  return await fetch(
    `https://${DATACENTER}.api.mailchimp.com/3.0/lists/${LIST_ID}/members/${subscriberHash}/tags`,

    {
      body: JSON.stringify(data),
      headers: {
        Authorization: `apikey ${API_KEY}`,
        "Content-Type": "application/json",
      },
      method: "POST",
    }
  );
}
