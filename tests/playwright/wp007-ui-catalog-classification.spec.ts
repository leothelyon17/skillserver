import { expect, test, type APIRequestContext, type APIResponse, type Page } from "@playwright/test";

type CatalogItem = {
  id: string;
  name: string;
};

function catalogCard(page: Page, title: string) {
  return page.locator(".skill-card").filter({
    has: page.locator("h3", { hasText: title }),
  });
}

async function openHome(page: Page) {
  await page.goto("/");
  await expect(page.locator(".skill-card").first()).toBeVisible();
}

async function openTaxonomyManager(page: Page) {
  await page.getByRole("button", { name: "Options" }).click();
  await page.getByRole("button", { name: "Taxonomy Manager" }).click();

  const modal = page.locator(".fixed.inset-0:visible").filter({
    has: page.getByRole("heading", { name: "Taxonomy Manager" }),
  });
  await expect(modal).toBeVisible();
  return modal;
}

async function responseErrorBody(response: APIResponse) {
  return (await response.text().catch(() => "")) || "<no body>";
}

async function createTaxonomyObject(
  request: APIRequestContext,
  target: string,
  payload: Record<string, unknown>,
) {
  const response = await request.post(target, { data: payload });
  if (response.status() === 201 || response.status() === 409) {
    return;
  }
  throw new Error(`Failed POST ${target}: ${response.status()} ${await responseErrorBody(response)}`);
}

async function patchCatalogTaxonomy(
  request: APIRequestContext,
  itemID: string,
  payload: Record<string, unknown>,
) {
  const response = await request.patch(`/api/catalog/${encodeURIComponent(itemID)}/taxonomy`, {
    data: payload,
  });
  if (!response.ok()) {
    throw new Error(
      `Failed PATCH /api/catalog/:id/taxonomy for ${itemID}: ${response.status()} ${await responseErrorBody(response)}`,
    );
  }
}

async function findCatalogItemByID(request: APIRequestContext, id: string): Promise<CatalogItem> {
  const response = await request.get("/api/catalog");
  expect(response.ok()).toBeTruthy();

  const items = (await response.json()) as CatalogItem[];
  const found = items.find((item) => item.id === id);
  if (!found) {
    throw new Error(`Catalog item not found by id: ${id}`);
  }
  return found;
}

function uniqueSuffix() {
  return `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;
}

test.describe("WP-007 catalog classification and taxonomy manager UX", () => {
  test("shows classification badges and applies classification-state filters", async ({ page }) => {
    const suffix = uniqueSuffix();
    const domainID = `domain-wp007-${suffix}`;
    const subdomainID = `subdomain-wp007-${suffix}`;
    const tagID = `tag-wp007-${suffix}`;

    await createTaxonomyObject(page.request, "/api/catalog/taxonomy/domains", {
      domain_id: domainID,
      key: `wp007-domain-${suffix}`,
      name: `WP007 Domain ${suffix}`,
    });
    await createTaxonomyObject(page.request, "/api/catalog/taxonomy/subdomains", {
      subdomain_id: subdomainID,
      domain_id: domainID,
      key: `wp007-subdomain-${suffix}`,
      name: `WP007 Subdomain ${suffix}`,
    });
    await createTaxonomyObject(page.request, "/api/catalog/taxonomy/tags", {
      tag_id: tagID,
      key: `wp007-tag-${suffix}`,
      name: `WP007 Tag ${suffix}`,
    });

    const additive = await findCatalogItemByID(page.request, "skill:additive-skill");
    const legacy = await findCatalogItemByID(page.request, "skill:legacy-skill");

    await patchCatalogTaxonomy(page.request, additive.id, {
      primary_domain_id: domainID,
      primary_subdomain_id: subdomainID,
      tag_ids: [tagID],
      updated_by: "playwright-wp007",
    });
    await patchCatalogTaxonomy(page.request, legacy.id, {
      primary_domain_id: domainID,
      primary_subdomain_id: subdomainID,
      clear_tags: true,
      updated_by: "playwright-wp007",
    });

    await openHome(page);

    const additiveCard = catalogCard(page, "additive-skill");
    const legacyCard = catalogCard(page, "legacy-skill");
    const gitCard = catalogCard(page, "fixture-git/git-skill");

    await expect(additiveCard).toBeVisible();
    await expect(additiveCard.getByText(new RegExp(`Primary: WP007 Domain ${suffix}`))).toBeVisible();
    await expect(additiveCard.getByText("Partially Classified")).toHaveCount(0);
    await expect(legacyCard.getByText("Partially Classified")).toBeVisible();
    await expect(legacyCard.getByText("Missing Tags")).toBeVisible();
    await expect(gitCard.getByText("Unclassified")).toBeVisible();

    await page.getByRole("button", { name: "Taxonomy Filters" }).click();
    await expect(page.getByLabel("Missing Primary Domain")).toBeVisible();

    const unclassifiedResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog" &&
        url.searchParams.get("unclassified") === "true"
      );
    });
    await page.getByLabel("Unclassified").check();
    await expect((await unclassifiedResponse).ok()).toBeTruthy();
    await expect(gitCard).toBeVisible();
    await expect(legacyCard).toHaveCount(0);
    await expect(additiveCard).toHaveCount(0);

    const clearUnclassifiedResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog" &&
        !url.searchParams.has("unclassified")
      );
    });
    await page.getByLabel("Unclassified").uncheck();
    await expect((await clearUnclassifiedResponse).ok()).toBeTruthy();

    const missingTagsResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog" &&
        url.searchParams.get("missing_tags") === "true"
      );
    });
    await page.getByLabel("Missing Tags").check();
    await expect((await missingTagsResponse).ok()).toBeTruthy();
    await expect(legacyCard).toBeVisible();
    await expect(additiveCard).toHaveCount(0);

    const clearMissingTagsResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog" &&
        !url.searchParams.has("missing_tags")
      );
    });
    await page.getByLabel("Missing Tags").uncheck();
    await expect((await clearMissingTagsResponse).ok()).toBeTruthy();

    const missingPrimaryResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog" &&
        url.searchParams.get("missing_primary_domain") === "true"
      );
    });
    await page.getByLabel("Missing Primary Domain").check();
    await expect((await missingPrimaryResponse).ok()).toBeTruthy();
    await expect(gitCard).toBeVisible();
    await expect(legacyCard).toHaveCount(0);
    await expect(additiveCard).toHaveCount(0);

    const clearFiltersResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog" &&
        !url.searchParams.has("unclassified") &&
        !url.searchParams.has("missing_tags") &&
        !url.searchParams.has("missing_primary_domain")
      );
    });
    await page.getByRole("button", { name: "Clear Filters" }).click();
    await expect((await clearFiltersResponse).ok()).toBeTruthy();
    await expect(additiveCard).toBeVisible();
    await expect(legacyCard).toBeVisible();
    await expect(gitCard).toBeVisible();
  });

  test("shows delete preflight usage and blocks in-use taxonomy deletion", async ({ page }) => {
    const suffix = uniqueSuffix();
    const inUseTagID = `tag-wp007-in-use-${suffix}`;
    const unusedTagID = `tag-wp007-unused-${suffix}`;
    const inUseTagName = `WP007 In Use ${suffix}`;
    const unusedTagName = `WP007 Unused ${suffix}`;

    await createTaxonomyObject(page.request, "/api/catalog/taxonomy/tags", {
      tag_id: inUseTagID,
      key: `wp007-in-use-${suffix}`,
      name: inUseTagName,
    });
    await createTaxonomyObject(page.request, "/api/catalog/taxonomy/tags", {
      tag_id: unusedTagID,
      key: `wp007-unused-${suffix}`,
      name: unusedTagName,
    });

    const additive = await findCatalogItemByID(page.request, "skill:additive-skill");
    await patchCatalogTaxonomy(page.request, additive.id, {
      add_tag_ids: [inUseTagID],
      updated_by: "playwright-wp007",
    });

    await openHome(page);
    const managerModal = await openTaxonomyManager(page);
    await managerModal.getByRole("button", { name: "Tags" }).click();

    const inUseRow = managerModal.locator("tr").filter({ hasText: inUseTagID });
    await expect(inUseRow).toBeVisible();
    await inUseRow.getByRole("button", { name: "Delete" }).click();

    const confirmModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Delete Tag" }),
    });
    await expect(confirmModal).toBeVisible();
    await expect(confirmModal.getByText(`Delete tag "${inUseTagName}"?`)).toBeVisible();
    await expect(confirmModal.getByText("Assignments", { exact: true })).toBeVisible();
    await expect(confirmModal.getByText("Catalog Items", { exact: true })).toBeVisible();
    await expect(confirmModal.getByText(additive.id)).toBeVisible();
    await expect(
      confirmModal.getByText(
        "This taxonomy object is still referenced and cannot be deleted until those assignments are removed.",
      ),
    ).toBeVisible();
    await expect(confirmModal.getByRole("button", { name: "Delete Tag" })).toBeDisabled();
    await confirmModal.getByRole("button", { name: "Close" }).click();
    await expect(confirmModal).toHaveCount(0);

    const unusedRow = managerModal.locator("tr").filter({ hasText: unusedTagID });
    await expect(unusedRow).toBeVisible();
    await unusedRow.getByRole("button", { name: "Delete" }).click();

    const unusedModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Delete Tag" }),
    });
    await expect(unusedModal).toBeVisible();
    await expect(unusedModal.getByText(`Delete tag "${unusedTagName}"? This action cannot be undone.`)).toBeVisible();
    await expect(unusedModal.getByRole("button", { name: "Delete Tag" })).toBeEnabled();

    const deleteResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "DELETE" &&
        response.url().includes(`/api/catalog/taxonomy/tags/${unusedTagID}`),
    );
    await unusedModal.getByRole("button", { name: "Delete Tag" }).click();
    await expect((await deleteResponse).ok()).toBeTruthy();
    await expect(unusedRow).toHaveCount(0);
  });

  test("shows delete preflight usage and blocks in-use domain deletion", async ({ page }) => {
    const suffix = uniqueSuffix();
    const domainID = `domain-wp007-in-use-${suffix}`;
    const subdomainID = `subdomain-wp007-in-use-${suffix}`;
    const domainName = `WP007 Domain In Use ${suffix}`;

    await createTaxonomyObject(page.request, "/api/catalog/taxonomy/domains", {
      domain_id: domainID,
      key: `wp007-domain-in-use-${suffix}`,
      name: domainName,
    });
    await createTaxonomyObject(page.request, "/api/catalog/taxonomy/subdomains", {
      subdomain_id: subdomainID,
      domain_id: domainID,
      key: `wp007-subdomain-in-use-${suffix}`,
      name: `WP007 Subdomain In Use ${suffix}`,
    });

    const additive = await findCatalogItemByID(page.request, "skill:additive-skill");
    await patchCatalogTaxonomy(page.request, additive.id, {
      primary_domain_id: domainID,
      primary_subdomain_id: subdomainID,
      updated_by: "playwright-wp007",
    });

    await openHome(page);
    const managerModal = await openTaxonomyManager(page);
    await managerModal.getByRole("button", { name: "Domains", exact: true }).click();

    const domainRow = managerModal.locator("tr").filter({ hasText: domainID }).first();
    await expect(domainRow).toBeVisible();
    await domainRow.getByRole("button", { name: "Delete" }).click();

    const confirmModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Delete Domain" }),
    });
    await expect(confirmModal).toBeVisible();
    await expect(confirmModal.getByText(`Delete domain "${domainName}"?`)).toBeVisible();
    await expect(confirmModal.getByText("Assignments", { exact: true })).toBeVisible();
    await expect(confirmModal.getByText("Catalog Items", { exact: true })).toBeVisible();
    await expect(confirmModal.getByText(additive.id)).toBeVisible();
    await expect(
      confirmModal.getByText(
        "This taxonomy object is still referenced and cannot be deleted until those assignments are removed.",
      ),
    ).toBeVisible();
    await expect(confirmModal.getByRole("button", { name: "Delete Domain" })).toBeDisabled();
    await confirmModal.getByRole("button", { name: "Close" }).click();
    await expect(confirmModal).toHaveCount(0);
  });

  test("surfaces taxonomy usage fetch errors without opening delete confirmation", async ({ page }) => {
    const suffix = uniqueSuffix();
    const domainID = `domain-wp007-error-${suffix}`;

    await createTaxonomyObject(page.request, "/api/catalog/taxonomy/domains", {
      domain_id: domainID,
      key: `wp007-domain-error-${suffix}`,
      name: `WP007 Domain Error ${suffix}`,
    });

    await openHome(page);
    const managerModal = await openTaxonomyManager(page);
    await managerModal.getByRole("button", { name: "Domains", exact: true }).click();

    await page.route(`**/api/catalog/taxonomy/domains/${domainID}/usage?*`, async (route) => {
      await route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({ error: "forced domain usage failure" }),
      });
    });

    const domainRow = managerModal.locator("tr").filter({ hasText: domainID });
    await expect(domainRow).toBeVisible();

    const usageResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "GET" &&
        response.url().includes(`/api/catalog/taxonomy/domains/${domainID}/usage`),
    );
    await domainRow.getByRole("button", { name: "Delete" }).click();
    expect((await usageResponse).status()).toBe(503);

    await expect(managerModal.getByText("forced domain usage failure")).toBeVisible();
    const confirmModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Delete Domain" }),
    });
    await expect(confirmModal).toHaveCount(0);
  });

  test("supports paginated metadata-first catalog responses and preview fallback reads", async ({ page }) => {
    const promptItem = {
      id: "prompt:mock-skill/prompts/mock-system.md",
      name: "mock-system.md",
      classifier: "prompt",
      description: "Preview prompt item without inline content.",
      parent_skill_id: "mock-skill",
      resource_path: "prompts/mock-system.md",
      has_assignment: false,
      is_fully_classified: false,
      missing_fields: ["primary_domain", "tags"],
    };
    const ruleItem = {
      id: "rule:mock-skill/rules/mock-policy.md",
      name: "mock-policy.md",
      classifier: "rule",
      description: "Mock rule item without inline content.",
      parent_skill_id: "mock-skill",
      resource_path: "rules/mock-policy.md",
      has_assignment: false,
      is_fully_classified: false,
      missing_fields: ["primary_domain", "tags"],
    };

    await page.route("**/api/catalog/search?*", async (route) => {
      const url = new URL(route.request().url());
      if (url.searchParams.get("q") !== "preview") {
        await route.continue();
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          items: [promptItem],
          next_cursor: "",
          has_more: false,
        }),
      });
    });

    await page.route("**/api/catalog?*", async (route) => {
      const url = new URL(route.request().url());
      const cursor = url.searchParams.get("cursor") || "";
      const payload =
        cursor === "cursor-2"
          ? {
              items: [ruleItem],
              next_cursor: "",
              has_more: false,
            }
          : {
              items: [promptItem],
              next_cursor: "cursor-2",
              has_more: true,
            };

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(payload),
      });
    });

    await page.route("**/api/skills/mock-skill/resources/prompts/mock-system.md", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "text/plain",
        body: "# Mock Prompt\n\nPreview content loaded on demand.",
      });
    });

    await openHome(page);

    const promptCard = catalogCard(page, "mock-system.md");
    const ruleCard = catalogCard(page, "mock-policy.md");

    await expect(promptCard).toBeVisible();
    await expect(page.getByText(/Page 1/)).toBeVisible();
    await expect(page.getByText(/more available/)).toBeVisible();

    const nextPageResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog" &&
        url.searchParams.get("cursor") === "cursor-2"
      );
    });
    await page.getByRole("button", { name: "Next" }).click();
    await expect((await nextPageResponse).ok()).toBeTruthy();
    await expect(ruleCard).toBeVisible();
    await expect(promptCard).toHaveCount(0);
    await expect(page.getByText(/Page 2/)).toBeVisible();

    const previousPageResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog" &&
        !url.searchParams.has("cursor")
      );
    });
    await page.getByRole("button", { name: "Previous" }).click();
    await expect((await previousPageResponse).ok()).toBeTruthy();
    await expect(promptCard).toBeVisible();

    const searchResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return (
        response.request().method() === "GET" &&
        url.pathname === "/api/catalog/search" &&
        url.searchParams.get("q") === "preview"
      );
    });
    await page.fill("#search-input", "preview");
    await expect((await searchResponse).ok()).toBeTruthy();
    await expect(promptCard).toBeVisible();

    const previewResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "GET" &&
        response.url().includes("/api/skills/mock-skill/resources/prompts/mock-system.md"),
    );
    await promptCard.getByRole("button", { name: "View" }).click();
    await expect((await previewResponse).ok()).toBeTruthy();

    const previewModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByText("Prompt catalog entries are read-only."),
    });
    await expect(previewModal).toBeVisible();
    await expect(previewModal.locator("textarea")).toHaveValue("# Mock Prompt\n\nPreview content loaded on demand.");
  });
});
