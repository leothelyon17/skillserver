import { expect, test, type Page } from "@playwright/test";

function catalogCard(page: Page, title: string) {
  return page.locator(".skill-card").filter({
    has: page.locator("h3", { hasText: title }),
  });
}

async function openHome(page: Page) {
  await page.goto("/");
  await expect(page.locator(".skill-card").first()).toBeVisible();
}

async function openMetadataModal(page: Page, title: string) {
  const card = catalogCard(page, title);
  await expect(card).toBeVisible();
  await card.first().getByRole("button", { name: "Metadata" }).click();

  const modal = page.locator(".fixed.inset-0:visible").filter({
    has: page.getByRole("heading", { name: "Edit Catalog Metadata" }),
  });
  await expect(modal).toBeVisible();
  await expect(modal.locator('[x-model="metadataEditorModal.form.displayName"]')).toBeVisible();
  return modal;
}

function waitForMetadataPatch(page: Page) {
  return page.waitForResponse(
    (response) =>
      response.request().method() === "PATCH" &&
      response.url().includes("/api/catalog/") &&
      response.url().endsWith("/metadata"),
  );
}

test.describe("WP-008 metadata overlay editing and mutability UX", () => {
  test("gates content actions for git-backed items while keeping metadata editable", async ({ page }) => {
    await openHome(page);

    const gitCard = catalogCard(page, "fixture-git/git-skill");
    await expect(gitCard).toBeVisible();
    await expect(gitCard.getByText("Read-only")).toBeVisible();
    await expect(gitCard.getByRole("button", { name: "Delete" })).toHaveCount(0);
    await expect(gitCard.getByRole("button", { name: "Metadata" })).toBeVisible();

    await gitCard.first().click();
    await expect(page.getByRole("heading", { name: "Edit Skill" })).toBeVisible();
    await expect(page.getByText("Read-only:")).toBeVisible();
    await expect(page.getByRole("button", { name: /^Save$/ })).toHaveCount(0);

    await page.keyboard.press("Escape");
    await expect(page.getByRole("heading", { name: "Edit Skill" })).toHaveCount(0);

    const metadataModal = await openMetadataModal(page, "fixture-git/git-skill");
    await expect(metadataModal.getByText("Content editing is locked for this catalog item.")).toBeVisible();
    await metadataModal.getByRole("button", { name: "Cancel" }).click();
  });

  test("keeps metadata modal open when metadata API is unavailable", async ({ page }) => {
    await page.route("**/api/catalog/*/metadata", async (route) => {
      if (route.request().method() !== "GET") {
        await route.continue();
        return;
      }

      await route.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({
          error: "catalog metadata API is unavailable",
        }),
      });
    });

    await openHome(page);

    const card = catalogCard(page, "additive-skill");
    await expect(card).toBeVisible();
    await card.first().getByRole("button", { name: "Metadata" }).click();

    const modal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Edit Catalog Metadata" }),
    });

    await expect(modal).toBeVisible();
    await expect(
      modal.getByText("Failed to load metadata: catalog metadata API is unavailable"),
    ).toBeVisible();
    await expect(modal.getByRole("button", { name: "Save Metadata" })).toBeDisabled();
  });

  test("saves git-backed metadata overlays and keeps them visible after reload/search", async ({ page }) => {
    await openHome(page);

    const displayName = "Git Skill Overlay UI";
    const description = "Metadata overlay updated from Playwright e2e";
    const labels = "git-overlay, ui-e2e";

    const modal = await openMetadataModal(page, "fixture-git/git-skill");

    await modal.locator('[x-model="metadataEditorModal.form.displayName"]').fill(displayName);
    await modal.locator('[x-model="metadataEditorModal.form.description"]').fill(description);
    await modal.locator('[x-model="metadataEditorModal.form.labelsText"]').fill(labels);
    await modal
      .locator('[x-model="metadataEditorModal.form.customMetadataJSON"]')
      .fill('{"owner":"playwright","priority":1}');

    const patchResponse = waitForMetadataPatch(page);
    await modal.getByRole("button", { name: "Save Metadata" }).click();
    await expect((await patchResponse).ok()).toBeTruthy();

    await expect(catalogCard(page, displayName)).toBeVisible();
    await expect(catalogCard(page, displayName).getByText("ui-e2e")).toBeVisible();

    await page.reload();
    await expect(catalogCard(page, displayName)).toBeVisible();
    await expect(catalogCard(page, displayName).getByText("ui-e2e")).toBeVisible();

    await page.fill("#search-input", displayName);
    await expect(catalogCard(page, displayName)).toBeVisible();
    await expect(page.locator(".skill-card")).toHaveCount(1);
  });

  test("shows validation errors for invalid metadata JSON and allows recovery", async ({ page }) => {
    await openHome(page);

    const updatedName = "additive-skill metadata overlay";
    const modal = await openMetadataModal(page, "additive-skill");

    await modal.locator('[x-model="metadataEditorModal.form.displayName"]').fill(updatedName);
    await modal.locator('[x-model="metadataEditorModal.form.customMetadataJSON"]').fill("[]");
    await modal.getByRole("button", { name: "Save Metadata" }).click();

    await expect(modal.getByText("Custom metadata must be a JSON object.")).toBeVisible();

    await modal
      .locator('[x-model="metadataEditorModal.form.customMetadataJSON"]')
      .fill('{"owner":"local-operator","priority":"high"}');

    const patchResponse = waitForMetadataPatch(page);
    await modal.getByRole("button", { name: "Save Metadata" }).click();
    await expect((await patchResponse).ok()).toBeTruthy();

    await expect(catalogCard(page, updatedName)).toBeVisible();
  });
});
