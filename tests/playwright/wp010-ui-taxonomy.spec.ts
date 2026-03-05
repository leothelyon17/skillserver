import { expect, test, type APIRequestContext, type APIResponse, type Page } from "@playwright/test";

type CatalogItem = {
  id: string;
  name: string;
};

async function openHome(page: Page) {
  await page.goto("/");
  await expect(page.locator(".skill-card").first()).toBeVisible();
}

function catalogCard(page: Page, title: string) {
  return page.locator(".skill-card").filter({
    has: page.locator("h3", { hasText: title }),
  });
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

async function seedTaxonomyAndAssignments(request: APIRequestContext) {
  await createTaxonomyObject(request, "/api/catalog/taxonomy/domains", {
    domain_id: "domain-platform",
    key: "platform",
    name: "Platform",
  });
  await createTaxonomyObject(request, "/api/catalog/taxonomy/domains", {
    domain_id: "domain-observability",
    key: "observability",
    name: "Observability",
  });

  await createTaxonomyObject(request, "/api/catalog/taxonomy/subdomains", {
    subdomain_id: "subdomain-platform-api",
    domain_id: "domain-platform",
    key: "api",
    name: "API",
  });
  await createTaxonomyObject(request, "/api/catalog/taxonomy/subdomains", {
    subdomain_id: "subdomain-observability-metrics",
    domain_id: "domain-observability",
    key: "metrics",
    name: "Metrics",
  });

  await createTaxonomyObject(request, "/api/catalog/taxonomy/tags", {
    tag_id: "tag-backend",
    key: "backend",
    name: "Backend",
  });
  await createTaxonomyObject(request, "/api/catalog/taxonomy/tags", {
    tag_id: "tag-metrics",
    key: "metrics",
    name: "Metrics",
  });

  const additive = await findCatalogItemByID(request, "skill:additive-skill");
  const legacy = await findCatalogItemByID(request, "skill:legacy-skill");

  await patchCatalogTaxonomy(request, additive.id, {
    primary_domain_id: "domain-platform",
    primary_subdomain_id: "subdomain-platform-api",
    secondary_domain_id: "domain-observability",
    secondary_subdomain_id: "subdomain-observability-metrics",
    tag_ids: ["tag-backend", "tag-metrics"],
    updated_by: "playwright-wp010",
  });

  await patchCatalogTaxonomy(request, legacy.id, {
    primary_domain_id: "domain-observability",
    primary_subdomain_id: "subdomain-observability-metrics",
    tag_ids: ["tag-metrics"],
    updated_by: "playwright-wp010",
  });
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

async function confirmDangerModal(page: Page) {
  const modal = page.locator(".fixed.inset-0:visible").filter({
    has: page.getByRole("button", { name: "Confirm" }),
  });
  await expect(modal).toBeVisible();
  await modal.getByRole("button", { name: "Confirm" }).click();
}

test.describe("WP-010 taxonomy manager and assignment UX", () => {
  test("manages taxonomy objects from the Options taxonomy manager", async ({ page }) => {
    await openHome(page);

    await page.getByRole("button", { name: "Options" }).click();
    await page.getByRole("button", { name: "Taxonomy Manager" }).click();

    const managerModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Taxonomy Manager" }),
    });
    await expect(managerModal).toBeVisible();

    const suffix = `${Date.now()}`.slice(-6);
    const domainID = `domain-wp010-${suffix}`;
    const subdomainID = `subdomain-wp010-${suffix}`;
    const tagID = `tag-wp010-${suffix}`;

    await managerModal.locator('[x-model="taxonomyManagerModal.forms.domain.domainID"]').fill(domainID);
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.domain.key"]').fill(`wp010-${suffix}`);
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.domain.name"]').fill(`WP010 Domain ${suffix}`);

    const createDomainResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        response.url().includes("/api/catalog/taxonomy/domains"),
    );
    await managerModal.getByRole("button", { name: "Create Domain" }).click();
    await expect((await createDomainResponse).ok()).toBeTruthy();

    const domainRow = managerModal.locator("tr").filter({ hasText: domainID });
    await expect(domainRow).toBeVisible();

    await domainRow.getByRole("button", { name: "Edit" }).click();
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.domain.name"]').fill(`WP010 Domain Updated ${suffix}`);

    const updateDomainResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "PATCH" &&
        response.url().includes(`/api/catalog/taxonomy/domains/${domainID}`),
    );
    await managerModal.getByRole("button", { name: "Update Domain" }).click();
    await expect((await updateDomainResponse).ok()).toBeTruthy();
    await expect(domainRow).toContainText(`WP010 Domain Updated ${suffix}`);

    await managerModal.getByRole("button", { name: "Subdomains" }).click();
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.subdomain.subdomainID"]').fill(subdomainID);
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.subdomain.domainID"]').selectOption(domainID);
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.subdomain.key"]').fill(`sub-wp010-${suffix}`);
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.subdomain.name"]').fill(`WP010 Subdomain ${suffix}`);

    const createSubdomainResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        response.url().includes("/api/catalog/taxonomy/subdomains"),
    );
    await managerModal.getByRole("button", { name: "Create Subdomain" }).click();
    await expect((await createSubdomainResponse).ok()).toBeTruthy();
    await expect(managerModal.locator("tr").filter({ hasText: subdomainID })).toBeVisible();

    await managerModal.getByRole("button", { name: "Tags" }).click();
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.tag.tagID"]').fill(tagID);
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.tag.key"]').fill(`tag-wp010-${suffix}`);
    await managerModal.locator('[x-model="taxonomyManagerModal.forms.tag.name"]').fill(`WP010 Tag ${suffix}`);

    const createTagResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        response.url().includes("/api/catalog/taxonomy/tags"),
    );
    await managerModal.getByRole("button", { name: "Create Tag" }).click();
    await expect((await createTagResponse).ok()).toBeTruthy();

    const tagRow = managerModal.locator("tr").filter({ hasText: tagID });
    await expect(tagRow).toBeVisible();

    const deleteTagResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "DELETE" &&
        response.url().includes(`/api/catalog/taxonomy/tags/${tagID}`),
    );
    await tagRow.getByRole("button", { name: "Delete" }).click();
    await confirmDangerModal(page);
    await expect((await deleteTagResponse).ok()).toBeTruthy();
    await expect(tagRow).toHaveCount(0);
  });

  test("assigns taxonomy in metadata editor and applies card/filter behavior", async ({ page }) => {
    await seedTaxonomyAndAssignments(page.request);
    await openHome(page);

    const additiveCard = catalogCard(page, "additive-skill");
    await expect(additiveCard).toBeVisible();
    await expect(additiveCard.getByText(/Primary: Platform/i)).toBeVisible();
    await expect(additiveCard.getByText(/Secondary: Observability/i)).toBeVisible();
    await expect(additiveCard.getByText("#Backend")).toBeVisible();

    const metadataModal = await openMetadataModal(page, "additive-skill");

    await expect(metadataModal.locator('[x-model="metadataEditorModal.form.primaryDomainID"]')).toHaveValue("domain-platform");

    const primarySubdomainSelect = metadataModal.locator('[x-model="metadataEditorModal.form.primarySubdomainID"]');
    await expect(primarySubdomainSelect.locator('option[value="subdomain-observability-metrics"]')).toHaveCount(0);

    await metadataModal.locator('[x-model="metadataEditorModal.form.primaryDomainID"]').selectOption("domain-observability");
    await expect(primarySubdomainSelect).toHaveValue("");
    await expect(primarySubdomainSelect.locator('option[value="subdomain-platform-api"]')).toHaveCount(0);

    await metadataModal.locator('[x-model="metadataEditorModal.form.primaryDomainID"]').selectOption("domain-platform");
    await metadataModal.locator('[x-model="metadataEditorModal.form.primarySubdomainID"]').selectOption("subdomain-platform-api");
    await metadataModal.locator('[x-model="metadataEditorModal.form.secondaryDomainID"]').selectOption("domain-observability");
    await metadataModal.locator('[x-model="metadataEditorModal.form.secondarySubdomainID"]').selectOption("subdomain-observability-metrics");

    const updatedName = "additive-skill wp010 taxonomy";
    await metadataModal.locator('[x-model="metadataEditorModal.form.displayName"]').fill(updatedName);

    const taxonomyPatchResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "PATCH" &&
        response.url().includes("/api/catalog/") &&
        response.url().endsWith("/taxonomy"),
    );
    const metadataPatchResponse = page.waitForResponse(
      (response) =>
        response.request().method() === "PATCH" &&
        response.url().includes("/api/catalog/") &&
        response.url().endsWith("/metadata"),
    );

    await metadataModal.getByRole("button", { name: "Save Metadata" }).click();
    await expect((await taxonomyPatchResponse).ok()).toBeTruthy();
    await expect((await metadataPatchResponse).ok()).toBeTruthy();

    const updatedCard = catalogCard(page, updatedName);
    await expect(updatedCard).toBeVisible();
    await expect(updatedCard.getByText("#Backend")).toBeVisible();
    await expect(updatedCard.getByText("#Metrics")).toBeVisible();

    await page.getByRole("button", { name: "Taxonomy Filters" }).click();
    await page.locator('[x-model="taxonomyFilters.primaryDomainID"]').selectOption("domain-platform");
    await expect(updatedCard).toBeVisible();
    await expect(catalogCard(page, "legacy-skill")).toHaveCount(0);

    await page.locator('[x-model="taxonomyFilters.primaryDomainID"]').selectOption("");

    await page
      .locator('label:has-text("Backend") input[type="checkbox"]')
      .first()
      .setChecked(true);
    await page
      .locator('label:has-text("Metrics") input[type="checkbox"]')
      .first()
      .setChecked(true);

    await page.locator('[x-model="taxonomyFilters.tagMatch"]').selectOption("all");
    await expect(updatedCard).toBeVisible();
    await expect(catalogCard(page, "legacy-skill")).toHaveCount(0);

    await page.locator('[x-model="taxonomyFilters.tagMatch"]').selectOption("any");
    await expect(updatedCard).toBeVisible();
    await expect(catalogCard(page, "legacy-skill")).toBeVisible();

    await page.getByRole("button", { name: "Clear Filters" }).click();

    const promptCard = catalogCard(page, "system.md");
    await expect(promptCard).toBeVisible();
    await promptCard.first().click();

    const promptModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByText("Prompt catalog entries are read-only."),
    });
    await expect(promptModal).toBeVisible();
    await promptModal.getByRole("button", { name: "Cancel" }).click();

    await updatedCard.first().click();
    await expect(page.getByRole("heading", { name: "Edit Skill" })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("heading", { name: "Edit Skill" })).toHaveCount(0);
  });
});
