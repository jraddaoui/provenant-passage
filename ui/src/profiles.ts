import type { ProvenantAipProfileEnum } from "@/openapi-generator";

export const profiles: {
  value: ProvenantAipProfileEnum;
  label: string;
  help: string;
}[] = [
  {
    value: "files_and_metadata",
    label: "Files and preservation metadata",
    help: "Extract the digital files and keep the AIP's original METS metadata.",
  },
  {
    value: "files_metadata_and_manifests",
    label: "Files, metadata and original checksum files",
    help: "Keep the extracted files and METS metadata, plus the AIP's original checksum files (BagIt manifests).",
  },
  {
    value: "original_and_extracted",
    label: "Original AIP and extracted content",
    help: "Keep the complete uploaded archive together with an extracted view of its files and METS metadata. Uses more storage space.",
  },
];
export const profileLabel = (value: ProvenantAipProfileEnum) =>
  profiles.find((p) => p.value === value)?.label ?? value;
