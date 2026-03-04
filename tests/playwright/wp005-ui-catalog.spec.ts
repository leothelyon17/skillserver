import { expect, test, type Page } from "@playwright/test";

async function openHome(page: Page) {
  await page.goto("/");
  await expect(page.locator(".skill-card").first()).toBeVisible();
}

function catalogCard(page: Page, title: string) {
  return page.locator(".skill-card").filter({
    has: page.locator("h3", { hasText: title }),
  });
}

async function confirmDangerModal(page: Page) {
  const modal = page.locator(".fixed.inset-0:visible").filter({
    has: page.getByRole("button", { name: "Confirm" }),
  });
  await expect(modal).toBeVisible();
  await modal.getByRole("button", { name: "Confirm" }).click();
}

test.describe("WP-005 unified catalog rendering and actions", () => {
  test("renders mixed catalog badges and opens prompts as read-only", async ({ page }) => {
    await openHome(page);

    const legacySkillCard = catalogCard(page, "legacy-skill");
    const additiveSkillCard = catalogCard(page, "additive-skill");
    const promptCard = catalogCard(page, "system.md");

    await expect(legacySkillCard).toBeVisible();
    await expect(additiveSkillCard).toBeVisible();
    await expect(promptCard).toBeVisible();

    await expect(legacySkillCard.locator(".catalog-classifier-badge")).toHaveText("skill");
    await expect(additiveSkillCard.locator(".catalog-classifier-badge")).toHaveText("skill");
    await expect(promptCard.locator(".catalog-classifier-badge")).toHaveText("prompt");

    await promptCard.first().click();

    const promptModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByText("Prompt catalog entries are read-only."),
    });
    await expect(promptModal).toBeVisible();
    await expect(promptModal.getByText("Open the parent skill to edit: additive-skill")).toBeVisible();
    await expect(promptModal.locator("textarea")).toHaveAttribute("readonly", /^(|readonly)$/);
    await expect(promptModal.getByRole("button", { name: "Save" })).toHaveCount(0);

    await promptModal.getByRole("button", { name: "Cancel" }).click();
    await expect(promptModal).toHaveCount(0);
  });

  test("searches across mixed catalog results", async ({ page }) => {
    await openHome(page);

    await page.fill("#search-input", "helpful");

    const promptCard = catalogCard(page, "system.md");
    await expect(promptCard).toBeVisible();
    await expect(promptCard.locator(".catalog-classifier-badge")).toHaveText("prompt");
    await expect(page.locator(".skill-card")).toHaveCount(1);

    await page.fill("#search-input", "legacy");

    const legacySkillCard = catalogCard(page, "legacy-skill");
    await expect(legacySkillCard).toBeVisible();
    await expect(legacySkillCard.locator(".catalog-classifier-badge")).toHaveText("skill");
    await expect(page.locator(".skill-card")).toHaveCount(1);

    await page.fill("#search-input", "");
    await expect(catalogCard(page, "legacy-skill")).toBeVisible();
    await expect(catalogCard(page, "additive-skill")).toBeVisible();
    await expect(catalogCard(page, "system.md")).toBeVisible();
  });

  test("filters catalog by classifier checkboxes", async ({ page }) => {
    await openHome(page);

    const showSkills = page.getByLabel("Show Skills");
    const showPrompts = page.getByLabel("Show Prompts");

    await expect(showSkills).toBeChecked();
    await expect(showPrompts).toBeChecked();

    await showPrompts.uncheck();
    await expect(catalogCard(page, "system.md")).toHaveCount(0);
    await expect(catalogCard(page, "legacy-skill")).toBeVisible();
    await expect(catalogCard(page, "additive-skill")).toBeVisible();
    await expect(page.locator(".skill-card")).toHaveCount(2);

    await showSkills.uncheck();
    await expect(page.locator(".skill-card")).toHaveCount(0);

    await showPrompts.check();
    await expect(catalogCard(page, "system.md")).toBeVisible();
    await expect(page.locator(".skill-card")).toHaveCount(1);

    await page.fill("#search-input", "legacy");
    await expect(page.locator(".skill-card")).toHaveCount(0);

    await showSkills.check();
    await expect(catalogCard(page, "legacy-skill")).toBeVisible();
    await expect(page.locator(".skill-card")).toHaveCount(1);
  });

  test("keeps skill edit and create/delete flows stable", async ({ page }) => {
    await openHome(page);

    const additiveSkillCard = catalogCard(page, "additive-skill");
    await expect(additiveSkillCard).toBeVisible();
    await additiveSkillCard.first().click();

    await expect(page.getByRole("heading", { name: "Edit Skill" })).toBeVisible();

    const updatedDescription = "Additive skill fixture for UI regression tests (wp005 edit regression)";
    await page.locator('textarea[x-model="skillDescription"]').fill(updatedDescription);
    const updateRequest = page.waitForResponse(
      (response) =>
        response.request().method() === "PUT" &&
        response.url().includes("/api/skills/additive-skill"),
    );
    await page.getByRole("button", { name: /Save/ }).first().click();
    await expect((await updateRequest).ok()).toBeTruthy();

    await page.getByRole("button", { name: /Cancel/ }).first().click();
    await expect(page.getByRole("heading", { name: "Edit Skill" })).toHaveCount(0);

    await page.fill("#search-input", "wp005 edit regression");
    await expect(catalogCard(page, "additive-skill")).toBeVisible();
    await expect(catalogCard(page, "additive-skill").locator(".catalog-classifier-badge")).toHaveText("skill");

    await page.fill("#search-input", "");

    const tempSkillName = "wp005-temp-skill";
    await page.getByRole("button", { name: "New Skill" }).click();
    await page.locator('input[x-model="skillName"]').fill(tempSkillName);
    await page.locator('textarea[x-model="skillDescription"]').fill("Temporary skill for WP-005 create/delete regression checks.");
    await page.locator('textarea[x-model="skillContent"]').fill("# Temporary\n\nUsed by Playwright regression test.");
    const createRequest = page.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        response.url().endsWith("/api/skills"),
    );
    await page.getByRole("button", { name: /Save/ }).first().click();

    await expect((await createRequest).ok()).toBeTruthy();

    await page.getByRole("button", { name: /Cancel/ }).first().click();

    const tempSkillCard = catalogCard(page, tempSkillName);
    await expect(tempSkillCard).toBeVisible();
    await expect(tempSkillCard.locator(".catalog-classifier-badge")).toHaveText("skill");

    await page.fill("#search-input", "wp005-temp");
    await expect(tempSkillCard).toBeVisible();
    await expect(catalogCard(page, "legacy-skill")).toHaveCount(0);
    await page.fill("#search-input", "");

    await tempSkillCard.getByRole("button", { name: "Delete" }).click();
    const deleteRequest = page.waitForResponse(
      (response) =>
        response.request().method() === "DELETE" &&
        response.url().includes(`/api/skills/${tempSkillName}`),
    );
    await confirmDangerModal(page);

    await expect((await deleteRequest).ok()).toBeTruthy();
    await expect(catalogCard(page, tempSkillName)).toHaveCount(0);
  });
});
