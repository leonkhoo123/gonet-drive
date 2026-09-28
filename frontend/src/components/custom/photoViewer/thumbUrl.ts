import { type FileInterface } from "@/api/api-file";

/** Map a photo's fullsize URL to its thumbnail variant. */
export const thumbUrl = (f: FileInterface) => f.url.replace("/photo/play/", "/photo/thumbnail/");
