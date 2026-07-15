import { createServerFn } from "@tanstack/react-start";
import type { DpaFieldKey } from "@/data/ticketOptions";
import { isValidCommonPaperCountry } from "@/data/commonPaperCountries";
import { formatDpaBody } from "@/data/ticketOptions";
import { createPlainThread } from "@/data/plain";

const API_BASE = "https://api.commonpaper.com/v1";

type CompanyAddress = {
  street: string;
  city: string;
  state: string;
  zip: string;
  country: string;
};

export type DpaRequestInput = {
  companyLegalName: string;
  signatoryName: string;
  signatoryTitle: string;
  signatoryEmail: string;
  companyAddress: string;
  country: string;
};

export type CreateDpaDraftResult = {
  id: string;
  status?: string;
  agreementUrl?: string;
};

export type CreateDpaRequestInput = {
  user: {
    id: string;
    name?: string;
  };
  dpa: DpaRequestInput;
  attachmentIds?: Array<string>;
};

export type CreateDpaRequestResult = {
  success: boolean;
  threadId?: string;
  agreementId?: string;
  agreementUrl?: string;
  error?: string;
};

function requireEnv(name: string, value: string | undefined): string {
  if (!value) {
    throw new Error(`Missing required env var: ${name}`);
  }
  return value;
}

function toDpaRequest(fields: Record<DpaFieldKey, string>): DpaRequestInput {
  return {
    companyLegalName: fields.companyName.trim(),
    signatoryName: fields.signatoryName.trim(),
    signatoryTitle: fields.signatoryTitle.trim(),
    signatoryEmail: fields.signatoryEmail.trim(),
    companyAddress: fields.companyAddress.trim(),
    country: fields.country.trim(),
  };
}

function normalizeCountryCode(country: string): string {
  const code = country.trim();
  if (!isValidCommonPaperCountry(code)) {
    throw new Error("Please select a valid country from the list");
  }
  return code;
}

function toCompanyAddress(address: string, country: string): CompanyAddress {
  return {
    street: address,
    city: "",
    state: "",
    zip: "",
    country: normalizeCountryCode(country),
  };
}

async function commonPaperFetch<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const apiKey = requireEnv(
    "COMMONPAPER_API_KEY",
    process.env.COMMONPAPER_API_KEY,
  );

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      Authorization: `Bearer ${apiKey}`,
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });

  const text = await res.text();

  let json: unknown;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = text;
  }

  if (!res.ok) {
    console.error("Common Paper API error:", JSON.stringify(json, null, 2));
    throw new Error(
      `Common Paper request failed: ${res.status} ${res.statusText}`,
    );
  }

  return json as T;
}

type CommonPaperAgreementResponse = {
  id: string;
  status?: string;
  agreement_url?: string;
};

function buildDpaDraftPayload(
  request: DpaRequestInput,
): Record<string, unknown> {
  const ownerEmail = requireEnv(
    "COMMONPAPER_OWNER_EMAIL",
    process.env.COMMONPAPER_OWNER_EMAIL,
  );
  const signerEmail = requireEnv(
    "COMMONPAPER_SIGNER_EMAIL",
    process.env.COMMONPAPER_SIGNER_EMAIL,
  );
  const templateId = requireEnv(
    "COMMONPAPER_DPA_TEMPLATE_ID",
    process.env.COMMONPAPER_DPA_TEMPLATE_ID,
  );

  const companyAddress = toCompanyAddress(
    request.companyAddress,
    request.country,
  );
  const testAgreement = process.env.COMMONPAPER_TEST_AGREEMENT === "true";

  return {
    template_id: templateId,
    owner_email: ownerEmail,
    signer_email: signerEmail,
    draft: true,
    agreement: {
      recipient_organization: request.companyLegalName,
      recipient_name: request.signatoryName,
      recipient_title: request.signatoryTitle,
      recipient_email: request.signatoryEmail,
      recipient_notice_email_address: request.signatoryEmail,
      recipient_street_address: companyAddress.street,
      recipient_city: companyAddress.city,
      recipient_state: companyAddress.state,
      recipient_zip: companyAddress.zip,
      recipient_country: companyAddress.country,
      test_agreement: testAgreement,
      message:
        "Thanks for requesting Inngest's Data Processing Agreement. Please review when ready.",
      dpa_attributes: {
        underlying_agreement_type: "external",
        underlying_agreement_val:
          "The standard terms and conditions of Inngest, and any Service Agreements or contracts signed by both parties",
        underlying_agreement_id: null,
        include_governing_country_eu: true,
        include_governing_country_eu_val: "Denmark",
        include_governing_country_uk: true,
        include_governing_country_uk_val: "England and Wales",
        include_external_subprocessor_location: "https://trust.inngest.com/",
        data_importer_contact_name: "Daniel Farrelly",
        data_importer_contact_position: "CTO",
        data_importer_contact_details: "",
        data_exporter_contact_name: request.signatoryName,
        data_exporter_contact_position: request.signatoryTitle,
        data_exporter_contact_details: request.signatoryEmail,
        data_exporter_address_country: companyAddress.country,
        data_exporter_address_street_address: companyAddress.street,
        data_exporter_address_city: companyAddress.city,
        data_exporter_address_state: companyAddress.state,
        data_exporter_address_zipcode: companyAddress.zip,
      },
    },
  };
}

export async function createDpaDraft(
  request: DpaRequestInput,
): Promise<CreateDpaDraftResult> {
  const payload = buildDpaDraftPayload(request);

  const agreement = await commonPaperFetch<CommonPaperAgreementResponse>(
    "/agreements",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
  );

  return {
    id: agreement.id,
    status: agreement.status,
    agreementUrl: agreement.agreement_url,
  };
}

export const createDpaRequest = createServerFn({ method: "POST" })
  .inputValidator((data: CreateDpaRequestInput) => data)
  .handler(async ({ data }): Promise<CreateDpaRequestResult> => {
    try {
      const { user, dpa, attachmentIds } = data;
      const ticketBody = formatDpaBody({
        companyName: dpa.companyLegalName,
        signatoryName: dpa.signatoryName,
        signatoryTitle: dpa.signatoryTitle,
        signatoryEmail: dpa.signatoryEmail,
        companyAddress: dpa.companyAddress,
        country: dpa.country,
      });

      const plainResult = await createPlainThread({
        user,
        ticket: {
          type: "dpa",
          title: dpa.companyLegalName,
          body: ticketBody,
          attachmentIds,
        },
      });

      if (!plainResult.success) {
        return {
          success: false,
          error:
            plainResult.error ||
            "Failed to create support ticket for DPA request.",
        };
      }

      try {
        const draft = await createDpaDraft(dpa);

        return {
          success: true,
          threadId: plainResult.threadId,
          agreementId: draft.id,
          agreementUrl: draft.agreementUrl,
        };
      } catch (error) {
        console.error("Error creating Common Paper DPA draft:", error);
        return {
          success: false,
          threadId: plainResult.threadId,
          error:
            error instanceof Error
              ? `Support ticket created, but the DPA draft failed: ${error.message}`
              : "Support ticket created, but the DPA draft failed.",
        };
      }
    } catch (error) {
      console.error("Error creating DPA request:", error);
      return {
        success: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to create DPA request",
      };
    }
  });

export { toDpaRequest };
