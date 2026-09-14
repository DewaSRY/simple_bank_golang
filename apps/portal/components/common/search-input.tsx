import { debounce } from "es-toolkit";
import { Search } from "lucide-react";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@/components/ui/input-group";
import { ComponentProps, ReactNode } from "react";
import { cn } from "@/lib/utils";

interface Props extends ComponentProps<"div"> {
  search: string;
  onSearch: (value: string) => void;
  placeholder?: string;
  addonText?: string;
  addon?: boolean;
  icon?: ReactNode;
}

export function SearchInput({
  search,
  onSearch,
  placeholder,
  addonText,
  addon,
  className,
  icon,
}: Props) {
  const debouncedOnSearch = debounce(onSearch, 300);

  return (
    <InputGroup className={cn(className)}>
      <InputGroupInput
        placeholder={placeholder ?? "Search..."}
        value={search}
        onChange={(e) => debouncedOnSearch(e.target.value)}
      />
      <InputGroupAddon>{icon ?? <Search />}</InputGroupAddon>
      {addon && (
        <InputGroupAddon align="inline-end">
          {addonText ?? "12 results"}
        </InputGroupAddon>
      )}
    </InputGroup>
  );
}
