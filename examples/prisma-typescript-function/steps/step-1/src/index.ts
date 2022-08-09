import type { Args } from "./types";
import { PrismaClient } from './generated/client'
const prisma = new PrismaClient();
export async function run({ event }: Args) {
  // The email field is either `event.user.email` or the receipt email.
  //
  // Note that Inngest creates Typescript types for each event you receive.
  const email = event.user.email || event.data.data.object.receipt_email;
  // Find the user.
  const user = await prisma.user.findUnique({
    where: { email },
  });
  if (user === null) {
    // Return an error which indicates that this user was not found.  This marks
    // the function as errored and allows you to handle these edge cases.
    return { status: 404, error: "This email was not found", email };
  }
  const charge = await prisma.charge.create({
    data: {
      userId: user.id,
      externalId: event.data.data.object.id,
      amount: event.data.data.object.amount,
      createdAt: new Date(event.data.created),
    },
  });
  // Return both the user and charge.  These will be accessible by
  // future steps, if you add steps to this function.
  return {
    status: 200,
    body: {
      user,
      charge,
    }
  };
}
