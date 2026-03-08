import { expect, test, type Page } from "@playwright/test";

function catalogCard(page: Page, title: string) {
  return page.locator(".skill-card").filter({
    has: page.locator("h3", { hasText: title }),
  });
}

function firstRuleCard(page: Page) {
  return page
    .locator(".skill-card")
    .filter({
      has: page.locator(".catalog-classifier-badge", { hasText: "rule" }),
    })
    .first();
}

async function openHome(page: Page) {
  await page.goto("/");
  await expect(page.locator(".skill-card").first()).toBeVisible();
}

test.describe("WP-010 UI export/materialization UX", () => {
  test("keeps skill export route behavior stable", async ({ page }) => {
    let exportRequestCount = 0;
    await page.route("**/api/skills/export/**", async (route) => {
      exportRequestCount += 1;
      await route.fulfill({
        status: 200,
        contentType: "application/gzip",
        body: "fixture archive bytes",
      });
    });

    await openHome(page);
    const additiveCard = catalogCard(page, "additive-skill");
    await expect(additiveCard).toBeVisible();

    await additiveCard.getByRole("button", { name: "Export" }).click();
    await expect.poll(() => exportRequestCount).toBe(1);
    expect(exportRequestCount).toBe(1);
  });

  test("previews dry-run before writes and supports prompt/rule materialization flows", async ({ page }) => {
    const materializeRequests: Array<Record<string, unknown>> = [];

    await page.route("**/api/runtime/capabilities", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          git: { stored_credentials_enabled: false },
          catalog: {
            rules_enabled: true,
            rule_directory_allowlist: ["rule", "rules"],
            rule_filename_allowlist: ["agents.md", "rules.md"],
          },
          mcp: {
            materialization_enabled: true,
            allowed_destination_roots: ["/tmp/playwright-materialize"],
          },
        }),
      });
    });

    await page.route("**/api/catalog/materialize", async (route) => {
      const payload = (route.request().postDataJSON() ?? {}) as Record<string, unknown>;
      materializeRequests.push(payload);
      const itemIDs = Array.isArray(payload.item_ids) ? payload.item_ids : [];
      const itemID = String(itemIDs[0] ?? "");
      const classifier = itemID.startsWith("rule:") ? "rule" : "prompt";
      const destinationDir = String(payload.destination_dir ?? "");
      const dryRun = Boolean(payload.dry_run);
      const conflictPolicy = String(payload.conflict_policy ?? "error");
      const fileName = classifier === "rule" ? "AGENTS.md" : "system.md";

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          dry_run: dryRun,
          destination_dir: destinationDir,
          resolved_destination_dir: destinationDir,
          items: [
            {
              item_id: itemID,
              classifier,
              conflict_policy: conflictPolicy,
              status: dryRun ? "planned" : "written",
              files: [
                {
                  source_path: classifier === "rule" ? "rules/agents.md" : "prompts/system.md",
                  target_path: fileName,
                  resolved_path: `${destinationDir}/${fileName}`,
                  action: "create",
                  conflict_policy: conflictPolicy,
                  exists: false,
                  written: !dryRun,
                  bytes: 32,
                },
              ],
            },
          ],
        }),
      });
    });

    await openHome(page);

    const promptCard = catalogCard(page, "system.md");
    await expect(promptCard).toBeVisible();
    await promptCard.getByRole("button", { name: "Materialize" }).click();

    const modal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Materialize Catalog Item" }),
    });
    await expect(modal).toBeVisible();

    await modal.locator('[x-model="materializationModal.destinationDir"]').fill("/tmp/playwright-materialize/project-a");
    await modal.locator('[x-model="materializationModal.conflictPolicy"]').selectOption("overwrite");

    await modal.getByRole("button", { name: "Preview Dry-Run" }).click();
    await expect.poll(() => materializeRequests.length).toBe(1);
    expect(materializeRequests[0].dry_run).toBe(true);
    await expect(modal.getByText("No writes occur until you click Materialize.")).toBeVisible();

    await modal.getByRole("button", { name: "Materialize" }).click();
    await expect.poll(() => materializeRequests.length).toBe(2);
    expect(materializeRequests[1].dry_run).toBe(false);
    await expect(modal.getByText("written")).toBeVisible();

    await modal.getByRole("button", { name: "Cancel" }).click();
    await expect(modal).toHaveCount(0);

    const ruleCard = firstRuleCard(page);
    await expect(ruleCard).toBeVisible();
    await ruleCard.getByRole("button", { name: "Materialize" }).click();

    const ruleModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Materialize Catalog Item" }),
    });
    await expect(ruleModal).toBeVisible();
    await ruleModal.locator('[x-model="materializationModal.destinationDir"]').fill("/tmp/playwright-materialize/project-b");
    await ruleModal.getByRole("button", { name: "Preview Dry-Run" }).click();

    await expect.poll(() => materializeRequests.length).toBe(3);
    expect(materializeRequests[2].dry_run).toBe(true);
    const thirdRequestItemIDs = Array.isArray(materializeRequests[2].item_ids)
      ? materializeRequests[2].item_ids
      : [];
    expect(String(thirdRequestItemIDs[0] ?? "")).toContain("rule:");
  });

  test("disables prompt/rule write actions when materialization capability is unavailable", async ({ page }) => {
    let materializeAttemptCount = 0;

    await page.route("**/api/runtime/capabilities", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          git: { stored_credentials_enabled: false },
          catalog: {
            rules_enabled: true,
            rule_directory_allowlist: ["rule", "rules"],
            rule_filename_allowlist: ["agents.md", "rules.md"],
          },
          mcp: {
            materialization_enabled: false,
            allowed_destination_roots: [],
          },
        }),
      });
    });

    await page.route("**/api/catalog/materialize", async (route) => {
      materializeAttemptCount += 1;
      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "unexpected materialize request in disabled test" }),
      });
    });

    await openHome(page);

    const promptCard = catalogCard(page, "system.md");
    await expect(promptCard).toBeVisible();
    await expect(promptCard.getByRole("button", { name: "Materialize" })).toBeDisabled();
    await expect(promptCard.getByText("Materialization is disabled by runtime capability settings.")).toBeVisible();

    const ruleCard = firstRuleCard(page);
    await expect(ruleCard).toBeVisible();
    await expect(ruleCard.getByRole("button", { name: "Materialize" })).toBeDisabled();

    expect(materializeAttemptCount).toBe(0);
  });
});
