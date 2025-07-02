export {};

declare global {
  interface CustomJwtSessionClaims {
    accountId?: string;
    externalId?: string;
    orgName?: string;
    orgPublickMetadata?: {
      accountId?: string;
    };
    fullName?: string;
    orgHasImage?: boolean;
    orgImageUrl?: string;
    username?: string;
  }
}
