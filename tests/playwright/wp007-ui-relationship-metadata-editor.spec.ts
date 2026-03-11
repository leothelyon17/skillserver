import { expect, test, type APIRequestContext, type APIResponse, type Page } from "@playwright/test";

const SKILL_ITEM_ID = "skill:additive-skill";
const PROMPT_ITEM_ID = "prompt:additive-skill:prompts/system.md";
const RULE_ITEM_ID = "rule:additive-skill:rules/agents.md";

function catalogCard(page: Page, title: string) {
  return page.locator(".skill-card").filter({
    has: page.locator("h3", { hasText: title }),
  });
}

async function openHome(page: Page) {
  await page.goto("/");
  await expect(page.locator(".skill-card").first()).toBeVisible();
}

async function responseErrorBody(response: APIResponse) {
  return (await response.text().catch(() => "")) || "<no body>";
}

async function patchSkillRelationships(
  request: APIRequestContext,
  payload: Record<string, unknown>,
) {
  const response = await request.patch(`/api/catalog/${encodeURIComponent(SKILL_ITEM_ID)}/relationships`, {
    data: payload,
  });
  if (!response.ok()) {
    throw new Error(
      `Failed PATCH /api/catalog/:id/relationships for ${SKILL_ITEM_ID}: ${response.status()} ${await responseErrorBody(response)}`,
    );
  }
}

async function openMetadataModal(page: Page, title: string) {
  const card = catalogCard(page, title);
  await expect(card).toBeVisible();
  await card.first().getByRole("button", { name: "Metadata" }).click();

  const modal = page.locator(".fixed.inset-0:visible").filter({
    has: page.getByRole("heading", { name: "Edit Catalog Metadata" }),
  });
  await expect(modal).toBeVisible();
  return modal;
}

test.describe("WP-007 relationship metadata editor UX", () => {
  test("loads and saves skill relationship state from metadata modal", async ({ page }) => {
    await patchSkillRelationships(page.request, {
      prompt_item_id: null,
      rule_item_ids: [],
      updated_by: "playwright-wp007",
    });

    await openHome(page);

    const additiveCard = catalogCard(page, "additive-skill");
    await expect(additiveCard).toBeVisible();
    await expect(additiveCard.getByText("prompts/system.md")).toHaveCount(0);
    await expect(additiveCard.getByText("rules/agents.md")).toHaveCount(0);

    const modal = await openMetadataModal(page, "additive-skill");

    const promptSelect = modal.getByTestId("metadata-relationship-prompt-select");
    const ruleSelect = modal.getByTestId("metadata-relationship-rule-select");
    await expect(promptSelect).toBeVisible();
    await expect(ruleSelect).toBeVisible();
    await expect(promptSelect.locator("option")).toContainText([
      "No prompt relationship",
      "system.md (additive-skill",
    ]);

    await promptSelect.selectOption(PROMPT_ITEM_ID);
    await ruleSelect.selectOption([RULE_ITEM_ID]);

    const metadataPatchResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "PATCH" &&
        url.pathname.startsWith("/api/catalog/") &&
        url.pathname.endsWith("/metadata")
      );
    });
    const relationshipPatchResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "PATCH" &&
        url.pathname.startsWith("/api/catalog/") &&
        url.pathname.endsWith("/relationships")
      );
    });

    await modal.getByRole("button", { name: "Save Metadata" }).click();

    await expect((await metadataPatchResponse).ok()).toBeTruthy();
    const relationshipResponse = await relationshipPatchResponse;
    await expect(relationshipResponse.ok()).toBeTruthy();

    const relationshipPayload = relationshipResponse.request().postDataJSON() as {
      prompt_item_id?: string | null;
      rule_item_ids?: string[];
    };
    expect(relationshipPayload.prompt_item_id).toBe(PROMPT_ITEM_ID);
    expect(relationshipPayload.rule_item_ids).toEqual([RULE_ITEM_ID]);

    const metadataAfterSaveResponse = await page.request.get(
      `/api/catalog/metadata?item_id=${encodeURIComponent(SKILL_ITEM_ID)}`,
    );
    await expect(metadataAfterSaveResponse.ok()).toBeTruthy();
    const metadataAfterSave = (await metadataAfterSaveResponse.json()) as {
      relationships?: {
        prompt?: { id?: string } | null;
        rules?: Array<{ id?: string }>;
      };
    };
    expect(metadataAfterSave.relationships?.prompt?.id ?? null).toBe(PROMPT_ITEM_ID);
    expect((metadataAfterSave.relationships?.rules ?? []).map((item) => item.id)).toContain(RULE_ITEM_ID);

    await expect(modal).toHaveCount(0);

    const metadataAfterCloseResponse = await page.request.get(
      `/api/catalog/metadata?item_id=${encodeURIComponent(SKILL_ITEM_ID)}`,
    );
    await expect(metadataAfterCloseResponse.ok()).toBeTruthy();
    const metadataAfterClose = (await metadataAfterCloseResponse.json()) as {
      relationships?: {
        prompt?: { id?: string } | null;
      };
    };
    expect(metadataAfterClose.relationships?.prompt?.id ?? null).toBe(PROMPT_ITEM_ID);

    const reloadedModal = await openMetadataModal(page, "additive-skill");
    await expect(reloadedModal.getByTestId("metadata-relationship-prompt-select")).toHaveValue(PROMPT_ITEM_ID);
    await expect(reloadedModal.getByTestId("metadata-relationship-rule-select")).toHaveValues([RULE_ITEM_ID]);
    await reloadedModal.getByRole("button", { name: "Cancel" }).click();

    await expect(additiveCard.getByText("prompts/system.md")).toHaveCount(0);
    await expect(additiveCard.getByText("rules/agents.md")).toHaveCount(0);
  });

  test("shows reverse-associated skills as read-only for prompt and rule metadata views", async ({ page }) => {
    await patchSkillRelationships(page.request, {
      prompt_item_id: PROMPT_ITEM_ID,
      rule_item_ids: [RULE_ITEM_ID],
      updated_by: "playwright-wp007",
    });

    await openHome(page);

    const promptModal = await openMetadataModal(page, "system.md");
    await expect(
      promptModal.getByText("Relationship writes are skill-owned. This view is reverse-derived and read-only."),
    ).toBeVisible();
    await expect(promptModal.getByText("additive-skill (skill:additive-skill)")).toBeVisible();
    await expect(promptModal.getByTestId("metadata-relationship-prompt-select")).toHaveCount(0);
    await promptModal.getByRole("button", { name: "Cancel" }).click();

    const ruleModal = await openMetadataModal(page, "agents.md");
    await expect(
      ruleModal.getByText("Relationship writes are skill-owned. This view is reverse-derived and read-only."),
    ).toBeVisible();
    await expect(ruleModal.getByText("additive-skill (skill:additive-skill)")).toBeVisible();
    await expect(ruleModal.getByTestId("metadata-relationship-rule-select")).toHaveCount(0);
    await ruleModal.getByRole("button", { name: "Cancel" }).click();
  });

  test("keeps metadata modal state intact when relationship save fails", async ({ page }) => {
    await patchSkillRelationships(page.request, {
      prompt_item_id: null,
      rule_item_ids: [],
      updated_by: "playwright-wp007",
    });

    await page.route("**/api/catalog/**/relationships", async (route) => {
      if (route.request().method() !== "PATCH") {
        await route.continue();
        return;
      }
      await route.fulfill({
        status: 400,
        contentType: "application/json",
        body: JSON.stringify({
          error: "forced relationship save failure",
        }),
      });
    });

    await openHome(page);
    const modal = await openMetadataModal(page, "additive-skill");

    const updatedName = "additive-skill wp007 relationship failure";
    await modal.locator('[x-model="metadataEditorModal.form.displayName"]').fill(updatedName);
    await modal.getByTestId("metadata-relationship-prompt-select").selectOption(PROMPT_ITEM_ID);
    await modal.getByTestId("metadata-relationship-rule-select").selectOption([RULE_ITEM_ID]);

    const metadataPatchResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "PATCH" &&
        url.pathname.startsWith("/api/catalog/") &&
        url.pathname.endsWith("/metadata")
      );
    });
    const relationshipPatchResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "PATCH" &&
        url.pathname.startsWith("/api/catalog/") &&
        url.pathname.endsWith("/relationships")
      );
    });

    await modal.getByRole("button", { name: "Save Metadata" }).click();
    await expect((await metadataPatchResponse).ok()).toBeTruthy();
    expect((await relationshipPatchResponse).status()).toBe(400);

    await expect(modal.getByText("forced relationship save failure", { exact: false })).toBeVisible();
    await expect(modal).toBeVisible();
    await expect(modal.locator('[x-model="metadataEditorModal.form.displayName"]')).toHaveValue(updatedName);
    await expect(modal.getByTestId("metadata-relationship-prompt-select")).toHaveValue(PROMPT_ITEM_ID);
    await expect(modal.getByTestId("metadata-relationship-rule-select")).toHaveValues([RULE_ITEM_ID]);
  });
});
