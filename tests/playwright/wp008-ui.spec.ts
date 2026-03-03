import { expect, test, type Page } from "@playwright/test";

async function openSkillResources(page: Page, skillName: string) {
  await page.goto("/");

  const skillCard = page.locator(".skill-card").filter({
    has: page.locator("h3", { hasText: skillName }),
  });
  await expect(skillCard).toBeVisible();
  await skillCard.first().click();

  const resourcesTab = page.getByRole("button", { name: /Resources/ });
  await expect(resourcesTab).toBeVisible();
  await resourcesTab.click();

  await expect(page.locator(".resource-section").first()).toBeVisible();
}

test.describe("WP-008 UI regression checks", () => {
  test("legacy skill keeps only legacy resource groups", async ({ page }) => {
    await openSkillResources(page, "legacy-skill");

    const headings = page.locator(".resource-section-header h3 span");
    await expect(headings.filter({ hasText: "Scripts" })).toBeVisible();
    await expect(headings.filter({ hasText: "References" })).toBeVisible();
    await expect(headings.filter({ hasText: "Assets" })).toBeVisible();
    await expect(headings.filter({ hasText: "Prompts" })).toHaveCount(0);
    await expect(headings.filter({ hasText: "Imported" })).toHaveCount(0);
  });

  test("additive skill renders prompts/imported groups and locks imported content", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await openSkillResources(page, "additive-skill");

    const headings = page.locator(".resource-section-header h3 span");
    await expect(headings.filter({ hasText: "Scripts" })).toBeVisible();
    await expect(headings.filter({ hasText: "References" })).toBeVisible();
    await expect(headings.filter({ hasText: "Assets" })).toBeVisible();
    await expect(headings.filter({ hasText: "Prompts" })).toBeVisible();
    await expect(headings.filter({ hasText: "Imported" })).toBeVisible();

    const importedSection = page.locator(".resource-section").filter({
      has: page.locator(".resource-section-header h3 span", { hasText: "Imported" }),
    });
    await expect(importedSection).toBeVisible();

    await expect(importedSection.getByRole("button", { name: "Upload" })).toHaveCount(0);

    const importedResource = importedSection.locator(".resource-item").filter({ hasText: "context.md" });
    await expect(importedResource).toBeVisible();
    await expect(importedResource.getByText("imported")).toBeVisible();
    await expect(importedResource.getByText("Read only")).toBeVisible();
    await expect(importedResource.getByText("Locked")).toBeVisible();
    await expect(importedResource.getByRole("button")).toHaveCount(0);

    await importedResource.locator(".resource-item-info").click();

    const visibleModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByText("View context.md"),
    });
    await expect(visibleModal).toBeVisible();
    await expect(visibleModal.locator("textarea")).toHaveAttribute("readonly", /^(|readonly)$/);
    await expect(visibleModal.getByRole("button", { name: "Save" })).toHaveCount(0);
  });

  test("narrow viewport keeps additive resources readable without horizontal overflow", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await openSkillResources(page, "additive-skill");

    const hasHorizontalOverflow = await page.evaluate(() => {
      const rootOverflow = document.documentElement.scrollWidth > window.innerWidth + 1;
      const sectionOverflow = Array.from(document.querySelectorAll(".resource-section-header")).some(
        (el) => el.scrollWidth > el.clientWidth + 1,
      );
      const rowOverflow = Array.from(document.querySelectorAll(".resource-item")).some(
        (el) => el.scrollWidth > el.clientWidth + 1,
      );
      return rootOverflow || sectionOverflow || rowOverflow;
    });

    expect(hasHorizontalOverflow).toBeFalsy();

    const importedSection = page.locator(".resource-section").filter({
      has: page.locator(".resource-section-header h3 span", { hasText: "Imported" }),
    });
    await expect(importedSection).toBeVisible();
    await expect(importedSection.locator(".resource-item").filter({ hasText: "context.md" })).toBeVisible();
  });
});
