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
      className="bg-canvasBase text-basis border-muted placeholder-disabled outline-primary-moderate w-full rounded-lg border p-3 text-sm outline-2 outline-offset-2 transition-all focus:outline"
    />
  );
}
