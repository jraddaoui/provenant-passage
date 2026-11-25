import { expect, it, vi } from "vitest";

import { createUploader } from "@/upload";

it("freezes the selected profile for retries and unlocks only after removal", () => {
  let selection = "files_and_metadata" as
    "files_and_metadata" | "original_and_extracted";
  const lock = vi.fn();
  const { uppy, endpoint } = createUploader(
    () => selection,
    lock,
    () => {},
  );
  const id = uppy.addFile({
    name: "a.tar",
    type: "application/x-tar",
    data: new Blob(["bytes"]),
  });
  selection = "original_and_extracted";
  uppy.opts.onBeforeUpload?.(uppy.getState().files);
  expect(endpoint()).toContain("profile=original_and_extracted");
  selection = "files_and_metadata";
  expect(endpoint()).toContain("profile=original_and_extracted");
  uppy.opts.onBeforeUpload?.(uppy.getState().files);
  expect(endpoint()).toContain("profile=original_and_extracted");
  uppy.removeFile(id);
  expect(lock).toHaveBeenLastCalledWith(false);
  uppy.destroy();
});
