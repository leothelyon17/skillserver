import { expect, test, type Page } from "@playwright/test";

type GitRepo = {
  id: string;
  url: string;
  name: string;
  enabled: boolean;
  auth_mode: string;
  credential_source: string;
  has_credentials: boolean;
  stored_credentials_enabled: boolean;
  last_sync_status: string;
  last_sync_error?: string;
};

async function openHome(page: Page) {
  await page.goto("/");
  await expect(page.locator(".skill-card").first()).toBeVisible();
}

async function openGitReposModal(page: Page) {
  await page.getByRole("button", { name: "Git Repos" }).click();
  const modal = page.locator(".fixed.inset-0:visible").filter({
    has: page.getByRole("heading", { name: "Git Repositories" }),
  });
  await expect(modal).toBeVisible();
  return modal;
}

test.describe("WP-007 private repo credential workflows and masked status UX", () => {
  test("keeps public onboarding simple and submits URL-only payload", async ({ page }) => {
    await page.route("**/api/runtime/capabilities", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ git: { stored_credentials_enabled: false } }),
      });
    });

    await page.route("**/api/git-repos", async (route) => {
      if (route.request().method() === "GET") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify([]),
        });
        return;
      }

      await route.fulfill({
        status: 201,
        contentType: "application/json",
        body: JSON.stringify({
          id: "public-repo",
          url: "https://github.com/acme/public-repo.git",
          name: "public-repo",
          enabled: true,
          auth_mode: "none",
          credential_source: "none",
          has_credentials: false,
          stored_credentials_enabled: false,
          last_sync_status: "never",
          last_sync_error: "",
        }),
      });
    });

    await openHome(page);
    const modal = await openGitReposModal(page);

    await expect(modal.locator('select[x-model="newGitRepo.authMode"]')).toBeHidden();

    await modal
      .locator('input[x-model="newGitRepo.url"]')
      .fill("https://github.com/acme/public-repo.git");

    const requestPromise = page.waitForRequest(
      (request) => request.method() === "POST" && request.url().endsWith("/api/git-repos"),
    );
    await modal.locator('button:has-text("Add")').last().click();

    const request = await requestPromise;
    expect(request.postDataJSON()).toEqual({
      url: "https://github.com/acme/public-repo.git",
    });
  });

  test("covers auth mode/source switching and env/file validation", async ({ page }) => {
    let postedPayload: Record<string, unknown> | null = null;
    let postCount = 0;

    await page.route("**/api/runtime/capabilities", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ git: { stored_credentials_enabled: false } }),
      });
    });

    await page.route("**/api/git-repos", async (route) => {
      if (route.request().method() === "GET") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify([]),
        });
        return;
      }

      postCount += 1;
      postedPayload = requestJson(route.request().postData());
      await route.fulfill({
        status: 201,
        contentType: "application/json",
        body: JSON.stringify({
          id: "private-repo",
          url: "https://github.com/acme/private-repo.git",
          name: "private-repo",
          enabled: true,
          auth_mode: "https_basic",
          credential_source: "env",
          has_credentials: true,
          stored_credentials_enabled: false,
          last_sync_status: "never",
          last_sync_error: "",
        }),
      });
    });

    await openHome(page);
    const modal = await openGitReposModal(page);

    await modal
      .locator('input[x-model="newGitRepo.url"]')
      .fill("https://github.com/acme/private-repo.git");
    await modal
      .getByRole("button", { name: /Private repository authentication \(optional\)/ })
      .click();

    const modeSelect = modal.locator('select[x-model="newGitRepo.authMode"]');
    await expect(modeSelect).toBeVisible();

    await modeSelect.selectOption("ssh_key");
    await expect(modal.locator('input[x-model="newGitRepo.keyRef"]')).toBeVisible();
    await expect(modal.locator('input[x-model="newGitRepo.tokenRef"]')).toBeHidden();

    await modeSelect.selectOption("https_basic");
    await modal.locator('select[x-model="newGitRepo.credentialSource"]').selectOption("env");

    await modal.locator('button:has-text("Add")').last().click();
    await expect
      .poll(() => postCount, { timeout: 2000 })
      .toBe(0);

    await modal.locator('input[x-model="newGitRepo.usernameRef"]:visible').fill("PRIVATE_GIT_USER");
    await modal.locator('input[x-model="newGitRepo.passwordRef"]:visible').fill("PRIVATE_GIT_PASSWORD");

    await modal.locator('button:has-text("Add")').last().click();
    await expect.poll(() => postCount).toBe(1);
    await expect.poll(() => postedPayload).not.toBeNull();

    expect(postedPayload).toEqual({
      url: "https://github.com/acme/private-repo.git",
      auth: {
        mode: "https_basic",
        source: "env",
        username_ref: "PRIVATE_GIT_USER",
        password_ref: "PRIVATE_GIT_PASSWORD",
      },
    });
  });

  test("supports stored-secret edit flow with masked values and status/error rendering", async ({ page }) => {
    let repos: GitRepo[] = [
      {
        id: "stored-repo",
        url: "https://github.com/acme/stored-private.git",
        name: "stored-private",
        enabled: true,
        auth_mode: "https_token",
        credential_source: "stored",
        has_credentials: true,
        stored_credentials_enabled: true,
        last_sync_status: "failed",
        last_sync_error: "authentication failed",
      },
    ];

    let latestUpdatePayload: Record<string, unknown> | null = null;

    await page.route("**/api/runtime/capabilities", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ git: { stored_credentials_enabled: true } }),
      });
    });

    await page.route("**/api/git-repos", async (route) => {
      if (route.request().method() === "GET") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(repos),
        });
        return;
      }

      await route.fulfill({
        status: 201,
        contentType: "application/json",
        body: JSON.stringify(repos[0]),
      });
    });

    await page.route("**/api/git-repos/stored-repo", async (route) => {
      latestUpdatePayload = requestJson(route.request().postData());
      repos = [
        {
          ...repos[0],
          last_sync_status: "success",
          last_sync_error: "",
        },
      ];
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(repos[0]),
      });
    });

    await openHome(page);
    const modal = await openGitReposModal(page);

    await expect(modal.getByText("Sync: Failed")).toBeVisible();
    await expect(modal.getByText("authentication failed")).toBeVisible();

    await modal.getByTitle("Edit repository").click();
    const editModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Edit Repository" }),
    });
    await expect(editModal).toBeVisible();

    await expect(editModal.getByText("Stored credential is configured. Values are masked")).toBeVisible();
    const tokenInput = editModal.locator('input[x-model="editingGitRepo.storedToken"]');
    await expect(tokenInput).toHaveValue("");

    await tokenInput.fill("new-secret-token");
    await editModal.locator('button:has-text("Save")').last().click();

    await expect.poll(() => latestUpdatePayload).not.toBeNull();
    expect(latestUpdatePayload).toEqual({
      url: "https://github.com/acme/stored-private.git",
      enabled: true,
      stored_credential: {
        token: "new-secret-token",
      },
    });

    await modal.getByTitle("Edit repository").click();
    const reopenedEditModal = page.locator(".fixed.inset-0:visible").filter({
      has: page.getByRole("heading", { name: "Edit Repository" }),
    });
    await expect(reopenedEditModal.locator('input[x-model="editingGitRepo.storedToken"]')).toHaveValue("");
  });

  test("keeps sync/toggle/delete actions working", async ({ page }) => {
    let repos: GitRepo[] = [
      {
        id: "regression-repo",
        url: "https://github.com/acme/regression-repo.git",
        name: "regression-repo",
        enabled: true,
        auth_mode: "none",
        credential_source: "none",
        has_credentials: false,
        stored_credentials_enabled: false,
        last_sync_status: "never",
        last_sync_error: "",
      },
    ];

    let syncCalls = 0;
    let toggleCalls = 0;
    let deleteCalls = 0;

    await page.route("**/api/runtime/capabilities", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ git: { stored_credentials_enabled: false } }),
      });
    });

    await page.route("**/api/git-repos", async (route) => {
      if (route.request().method() === "GET") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(repos),
        });
        return;
      }

      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "unexpected" }),
      });
    });

    await page.route("**/api/git-repos/regression-repo/sync", async (route) => {
      syncCalls += 1;
      repos = [
        {
          ...repos[0],
          last_sync_status: "success",
          last_sync_error: "",
        },
      ];
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(repos[0]),
      });
    });

    await page.route("**/api/git-repos/regression-repo/toggle", async (route) => {
      toggleCalls += 1;
      repos = [
        {
          ...repos[0],
          enabled: !repos[0].enabled,
        },
      ];
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(repos[0]),
      });
    });

    await page.route("**/api/git-repos/regression-repo", async (route) => {
      deleteCalls += 1;
      repos = [];
      await route.fulfill({ status: 204 });
    });

    await openHome(page);
    const modal = await openGitReposModal(page);

    await modal.getByTitle("Sync repository").click();
    await expect.poll(() => syncCalls).toBe(1);
    await expect(modal.getByText("Sync: Success")).toBeVisible();

    await modal.getByTitle("Disable repository").click();
    await expect.poll(() => toggleCalls).toBe(1);
    await expect(modal.getByText("Disabled")).toBeVisible();

    await modal.getByTitle("Delete repository").click();
    await page.getByRole("button", { name: "Confirm" }).click();
    await expect.poll(() => deleteCalls).toBe(1);
    await expect(modal.getByText("No git repositories configured")).toBeVisible();
  });
});

function requestJson(postData: string | null): Record<string, unknown> {
  if (!postData) {
    return {};
  }
  return JSON.parse(postData) as Record<string, unknown>;
}
