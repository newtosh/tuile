/**
 * Structural checks for browser SVG exports that wrap a single raster image.
 * Used by node --test and scripts/verify-export-svg.sh.
 */

const SVG_ROOT_RE =
  /<svg[^>]*\bwidth="(\d+)"[^>]*\bheight="(\d+)"[^>]*\bviewBox="0 0 (\d+) (\d+)"/;
const SVG_IMAGE_RE =
  /<image\b[^>]*\bx="([^"]*)"[^>]*\by="([^"]*)"[^>]*\bwidth="([^"]*)"[^>]*\bheight="([^"]*)"[^>]*\bhref="data:image\/(png|jpeg);base64,([^"]+)"/;

export function pngDimensionsFromBase64(base64) {
  const buf = Buffer.from(base64, "base64");
  if (buf.length < 26 || buf.toString("ascii", 1, 4) !== "PNG") {
    throw new Error("invalid png payload");
  }
  return {
    width: buf.readUInt32BE(16),
    height: buf.readUInt32BE(20),
    colorType: buf[25],
  };
}

export function parseSvgRasterExport(svgText) {
  const root = SVG_ROOT_RE.exec(svgText);
  if (!root) {
    throw new Error("missing svg width/height/viewBox");
  }
  const image = SVG_IMAGE_RE.exec(svgText);
  if (!image) {
    throw new Error("missing embedded raster <image>");
  }
  return {
    width: Number(root[1]),
    height: Number(root[2]),
    viewBoxW: Number(root[3]),
    viewBoxH: Number(root[4]),
    imageX: Number(image[1]),
    imageY: Number(image[2]),
    imageW: Number(image[3]),
    imageH: Number(image[4]),
    rasterMime: image[5],
    rasterBase64: image[6],
  };
}

export function validateSvgRasterExport(parsed, svgText = "") {
  const errors = [];
  if (parsed.width !== parsed.viewBoxW || parsed.height !== parsed.viewBoxH) {
    errors.push("svg width/height must match viewBox");
  }
  if (parsed.imageX !== 0 || parsed.imageY !== 0) {
    errors.push("embedded image must be anchored at 0,0");
  }
  if (parsed.imageW !== parsed.width || parsed.imageH !== parsed.height) {
    errors.push("embedded image display size must fill the svg viewport");
  }
  if (!["png", "jpeg"].includes(parsed.rasterMime)) {
    errors.push(`unsupported embedded raster mime ${parsed.rasterMime}`);
  }
  if (parsed.rasterMime === "png") {
    try {
      const png = pngDimensionsFromBase64(parsed.rasterBase64);
      if (png.width < parsed.width || png.height < parsed.height) {
        errors.push(
          `embedded png is ${png.width}x${png.height}, expected at least ${parsed.width}x${parsed.height}`
        );
      }
      if (png.width % parsed.width !== 0 || png.height % parsed.height !== 0) {
        errors.push(
          `embedded png ${png.width}x${png.height} must be an integer multiple of svg ${parsed.width}x${parsed.height}`
        );
      }
    } catch (err) {
      errors.push(`embedded png invalid: ${err.message}`);
    }
  }
  if (/\btransform\s*=\s*["']scale\(/.test(svgText)) {
    errors.push("svg must not use nested scale transforms");
  }
  return errors;
}

export function assertValidSvgRasterExport(svgText) {
  const parsed = parseSvgRasterExport(svgText);
  const errors = validateSvgRasterExport(parsed, svgText);
  if (errors.length) {
    throw new Error(errors.join("; "));
  }
  return parsed;
}
