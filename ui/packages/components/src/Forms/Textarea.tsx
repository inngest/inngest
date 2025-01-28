export interface TextareaProps {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  rows?: number;
  required?: boolean;
}

export function Textarea({ value, onChange, placeholder, rows = 3, required }: TextareaProps) {
  return (
    <textarea
      placeholder={placeholder}
      value={value}
      onChange={(e) => {
        onChange(e.currentTarget.value);
      }}
      rows={rows}
      required={required}
      className="bg-canvasBase text-basis border-muted placeholder-disabled focus:outline-primary-moderate focus:border-muted w-full rounded-md border p-3 text-sm outline-2 transition-all focus:outline focus:ring-0"
    />
  );
}
