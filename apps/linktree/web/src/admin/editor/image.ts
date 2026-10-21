// Prepares an image in the browser before the upload: drawing it on a canvas
// applies the EXIF orientation of phone photos, and shrinking it to 1024 px keeps
// the upload small. The server checks and re-encodes it anyway.

export const maxUploadSide = 1024;

/** The size of a w×h image fitted within max×max, keeping its ratio. */
export function fitWithin(w: number, h: number, max: number): { width: number; height: number } {
  if (w <= max && h <= max) return { width: w, height: h };
  const scale = max / Math.max(w, h);
  return { width: Math.max(1, Math.round(w * scale)), height: Math.max(1, Math.round(h * scale)) };
}

export async function prepareImage(file: File): Promise<Blob> {
  // createImageBitmap applies the EXIF orientation ("from-image" is the default).
  const bitmap = await createImageBitmap(file);
  const { width, height } = fitWithin(bitmap.width, bitmap.height, maxUploadSide);
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("canvas unavailable");
  ctx.drawImage(bitmap, 0, 0, width, height);
  bitmap.close();
  // PNG keeps the transparency of logos; photos become JPEG.
  const type = file.type === "image/png" ? "image/png" : "image/jpeg";
  return new Promise((resolve, reject) =>
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error("encoding failed"))),
      type,
      0.9,
    ),
  );
}
