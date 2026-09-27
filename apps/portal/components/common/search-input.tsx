import { Search, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
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
  autoFocus?: boolean;
}

export function SearchInput({
  search,
  onSearch,
  placeholder,
  addonText,
  addon,
  className,
  icon,
  autoFocus,
}: Props) {
  const { t } = useTranslation("common");
  const resolvedPlaceholder = placeholder ?? t("searchPlaceholder");

  return (
    <InputGroup className={cn(className)}>
      <InputGroupInput
        type="search"
        aria-label={resolvedPlaceholder}
        placeholder={resolvedPlaceholder}
        value={search}
        autoFocus={autoFocus}
        onChange={(e) => onSearch(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Escape" && search) {
            e.stopPropagation();
            onSearch("");
          }
        }}
        className="[&::-webkit-search-cancel-button]:hidden"
      />
      <InputGroupAddon>{icon ?? <Search aria-hidden />}</InputGroupAddon>
      {search ? (
        <InputGroupAddon align="inline-end">
          <InputGroupButton
            size="icon-xs"
            aria-label={t("clearSearch")}
            onClick={() => onSearch("")}
          >
            <X aria-hidden />
          </InputGroupButton>
        </InputGroupAddon>
      ) : null}
      {addon && (
        <InputGroupAddon align="inline-end">{addonText}</InputGroupAddon>
      )}
    </InputGroup>
  );
}
