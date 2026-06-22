import { BoardMembersSection } from "@/components/board/members/BoardMembersSection";

import { use } from "react";

interface PageProps {
  params: Promise<{ boardId: string }>;
}

export default function OverviewPage({ params }: PageProps) {
  const { boardId } = use(params);
  return (
    <div className="-mx-4 -mb-8 min-h-full bg-slate-50 px-4 pb-12 pt-8 md:-mx-6 md:px-8 lg:-mx-8 lg:px-10">
      <div className="animate-in fade-in duration-200">
        <BoardMembersSection boardId={boardId} />
      </div>
    </div>
  );
}
