import Uppy from "@uppy/core";
import XHR from "@uppy/xhr-upload";

import { getPath } from "@/client";
import type { ProvenantAipProfileEnum } from "@/openapi-generator";

export function createUploader(
  selected: () => ProvenantAipProfileEnum,
  locked: (value: boolean) => void,
  accepted: () => void,
) {
  let frozen: ProvenantAipProfileEnum | null = null;
  const endpoint = () =>
    getPath() +
    "/aips/upload?profile=" +
    encodeURIComponent(frozen ?? selected());
  const uppy = new Uppy({
    restrictions: {
      maxNumberOfFiles: 1,
      minNumberOfFiles: 1,
      allowedFileTypes: [".tar", ".7z"],
    },
    onBeforeUpload: (files) => {
      frozen ??= selected();
      locked(true);
      return files;
    },
  }).use(XHR, {
    endpoint,
    allowedMetaFields: false,
    fieldName: "file",
    timeout: 0,
    // A failed HTTP response can represent an already accepted request. Never
    // transparently submit another copy; the UI retains its profile for a manual retry.
    shouldRetry: () => false,
    onAfterResponse: (xhr) => {
      if (xhr.status === 202) accepted();
    },
    getResponseData: (xhr) => {
      if (xhr.status !== 202) {
        let message = "Upload failed. Check the AIPs list before retrying.";
        try {
          const body = JSON.parse(xhr.responseText);
          if (typeof body.message === "string") message = body.message;
        } catch {
          /* Use fallback. */
        }
        throw new Error(message);
      }
      return { url: "" };
    },
  });
  uppy.on("file-removed", () => {
    if (uppy.getFiles().length === 0) {
      frozen = null;
      locked(false);
    }
  });
  return { uppy, endpoint };
}
