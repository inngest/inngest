export default function isValidUrl(string: string) {
  try {
    const newUrl = new URL(string);
    return true;
  } catch (err) {
    return false;
  }
}
