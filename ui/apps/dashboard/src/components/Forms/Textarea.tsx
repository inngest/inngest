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
      className="w-full rounded-lg border border-slate-300 px-3 py-3 text-sm placeholder-slate-500 shadow outline-2 outline-offset-2 outline-indigo-500 transition-all focus:outline"
    ></textarea>
  );
}
