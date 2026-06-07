'use client';

interface FormFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  autoComplete?: string;
}

export function FormField({
  label,
  value,
  onChange,
  type = 'text',
  required,
  minLength,
  maxLength,
  autoComplete,
}: FormFieldProps) {
  return (
    <label className="block text-sm">
      <span className="mb-1 block font-medium text-gray-700">{label}</span>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        required={required}
        minLength={minLength}
        maxLength={maxLength}
        autoComplete={autoComplete}
        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-gray-900 focus:border-gray-500 focus:outline-none"
      />
    </label>
  );
}

export function FormError({ message, details }: { message: string; details?: string[] }) {
  return (
    <div className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
      <p>{message}</p>
      {details && details.length > 0 && (
        <ul className="mt-1 list-inside list-disc">
          {details.map((d) => (
            <li key={d}>{d}</li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function FormSuccess({ message }: { message: string }) {
  return <div className="rounded-lg bg-green-50 p-3 text-sm text-green-700">{message}</div>;
}
