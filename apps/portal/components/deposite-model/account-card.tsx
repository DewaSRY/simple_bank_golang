import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
interface Props {
  name: string;
  number: string;
  selected: boolean;
  onClick: () => void;
}

export function AccountCard({
  name,
  number,

  selected,
  onClick,
}: Props) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "w-full! p-2 rounded-lg flex flex-col items-start border",
        selected && "border-blue-500 bg-blue-100",
      )}
    >
      <h3 className="text-gray-900 font-semibold">{name}</h3>
      <p className="text-sm text-gray-500">{number}</p>
    </button>
  );
}
